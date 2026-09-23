// internal/capabilities/queue/redis/redis_queue.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jimu/internal/capabilities/queue"

	redistore "jimu/internal/kernel/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func init() { queue.Register(queue.TypeRedis, New) }

// New 构造 Redis 队列驱动；连接由装配期注入（queue.Config.Redis），不来自配置解码。
func New(cfg queue.Config) (queue.Queue, error) { return NewRedisQueue(cfg.Redis), nil }

// visibilityTTL 可见性超时：任务被取走但未 Ack/Nack 时重新入队。
const visibilityTTL = 5 * time.Minute

// RedisQueue Redis 任务队列。
// 消费采用 at-least-once：Consume 原子移入 processing 列表并登记可见性超时，
// 处理成功 Ack、失败 Nack；超时未确认的任务由 RequeueExpired 重新入队。
// 每个 Consume 生成唯一 token 并写入 JobData，使 processing 列表中的元素可精确匹配，
// 避免过期重试与 Ack/Nack 误删同名任务。
type RedisQueue struct {
	client redistore.Client
}

// NewRedisQueue 创建 Redis 队列
func NewRedisQueue(client redistore.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

// Submit 提交任务到实时队列
func (q *RedisQueue) Submit(ctx context.Context, job *queue.JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.client.LPush(ctx, queue.QueueKey, data).Err()
}

// SubmitDelayed 提交延迟任务
func (q *RedisQueue) SubmitDelayed(ctx context.Context, job *queue.JobData, delay time.Duration) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	score := time.Now().Add(delay).Unix()
	return q.client.ZAdd(ctx, queue.DelayedKey, redis.Z{Score: float64(score), Member: data}).Err()
}

// Consume 消费任务。阻塞等待，成功后原子移入 processing 并登记可见性超时。
// processing 列表中保存带 token 的完整 JSON，使 Ack/Nack 可按 token 精确匹配。
func (q *RedisQueue) Consume(ctx context.Context, timeout time.Duration) (*queue.JobData, error) {
	res, err := q.client.BLMove(ctx, queue.QueueKey, queue.ProcessingKey, "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return nil, err
	}
	var job queue.JobData
	if err := json.Unmarshal([]byte(res), &job); err != nil {
		return nil, err
	}
	job.Token = uuid.NewString()
	job.Deadline = time.Now().Add(visibilityTTL).Unix()
	data, err := json.Marshal(&job)
	if err != nil {
		return nil, fmt.Errorf("marshal in-flight job: %w", err)
	}
	// processing 中更新为带 token 的版本，替换 BLMove 移入的原始 JSON
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, queue.ProcessingKey, 1, res)
	pipe.RPush(ctx, queue.ProcessingKey, data)
	pipe.ZAdd(ctx, queue.InFlightKey, redis.Z{
		Score:  float64(job.Deadline),
		Member: data,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	return &job, nil
}

// Ack 确认任务：从 processing 列表与 in-flight 登记中移除，任务完成。
func (q *RedisQueue) Ack(ctx context.Context, job *queue.JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, queue.ProcessingKey, 1, data)
	pipe.ZRem(ctx, queue.InFlightKey, data)
	_, err = pipe.Exec(ctx)
	return err
}

// Nack 否认任务：从 processing 移除并重新入队，供重试。
func (q *RedisQueue) Nack(ctx context.Context, job *queue.JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, queue.ProcessingKey, 1, data)
	pipe.ZRem(ctx, queue.InFlightKey, data)
	pipe.LPush(ctx, queue.QueueKey, data)
	_, err = pipe.Exec(ctx)
	return err
}

// RequeueExpired 将超时未确认的任务重新入队（返回重新入队数量）。
// 可见性超时兜底：worker 崩溃或处理超时后任务不会丢失。
func (q *RedisQueue) RequeueExpired(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	members, err := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     queue.InFlightKey,
		ByScore: true,
		Start:   "-inf",
		Stop:    fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, nil
	}
	pipe := q.client.Pipeline()
	for _, m := range members {
		pipe.LRem(ctx, queue.ProcessingKey, 1, m)
		pipe.ZRem(ctx, queue.InFlightKey, m)
		pipe.LPush(ctx, queue.QueueKey, m)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return len(members), nil
}

// MoveDueJobs 将到期的延迟任务移入实时队列
func (q *RedisQueue) MoveDueJobs(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	members, err := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     queue.DelayedKey,
		ByScore: true,
		Start:   "0",
		Stop:    fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, nil
	}
	pipe := q.client.Pipeline()
	for _, m := range members {
		pipe.LPush(ctx, queue.QueueKey, m)
		pipe.ZRem(ctx, queue.DelayedKey, m)
	}
	_, err = pipe.Exec(ctx)
	return len(members), err
}
