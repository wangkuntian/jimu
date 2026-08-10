package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	QueueKey   = "jimu:queue:default"
	DelayedKey = "jimu:queue:delayed"
)

// RedisQueue Redis 任务队列
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue 创建 Redis 队列
func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

// Submit 提交任务到实时队列
func (q *RedisQueue) Submit(ctx context.Context, job *JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.client.LPush(ctx, QueueKey, data).Err()
}

// SubmitDelayed 提交延迟任务
func (q *RedisQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	score := time.Now().Add(delay).Unix()
	return q.client.ZAdd(ctx, DelayedKey, redis.Z{Score: float64(score), Member: data}).Err()
}

// Consume 消费任务（阻塞式）
func (q *RedisQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	result, err := q.client.BRPop(ctx, timeout, QueueKey).Result()
	if err != nil {
		return nil, err
	}
	var job JobData
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// MoveDueJobs 将到期的延迟任务移入实时队列
func (q *RedisQueue) MoveDueJobs(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	members, err := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     DelayedKey,
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
		pipe.LPush(ctx, QueueKey, m)
		pipe.ZRem(ctx, DelayedKey, m)
	}
	_, err = pipe.Exec(ctx)
	return len(members), err
}

// Ack 确认任务。Redis 为破坏性消费，任务已在 Consume 时移除，无需 ack。
func (q *RedisQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack 否认任务。Redis 破坏性消费下任务已丢失，no-op 保持接口一致。
func (q *RedisQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// JobData Redis 中的任务数据
type JobData struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}
