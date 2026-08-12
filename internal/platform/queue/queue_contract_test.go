// internal/platform/queue/queue_contract_test.go
package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// newRedisTestQueue 创建基于 miniredis 的测试队列（无需外部 Redis）
func newRedisTestQueue(t *testing.T) (*RedisQueue, *redis.Client) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run failed: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewRedisQueue(client), client
}

func TestRedisQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RedisQueue)(nil)
	var _ Consumer = (*RedisQueue)(nil)
}

func TestQueueContract_SubmitConsume(t *testing.T) {
	rq, _ := newRedisTestQueue(t)
	var q Queue = rq
	var c Consumer = rq

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	got, err := c.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Payload, got.Payload)
	assert.NotEmpty(t, got.Token, "Consume 应生成唯一 token")
	assert.NotZero(t, got.Deadline, "Consume 应登记可见性超时")
}

func TestRedisQueueAckRemovesFromProcessing(t *testing.T) {
	rq, client := newRedisTestQueue(t)

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, rq.Submit(context.Background(), job))

	got, err := rq.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)

	// 处理中：processing 有 1 条，in_flight 有 1 条
	assert.Equal(t, int64(1), client.LLen(context.Background(), ProcessingKey).Val())
	assert.Equal(t, int64(1), client.ZCard(context.Background(), InFlightKey).Val())

	assert.NoError(t, rq.Ack(context.Background(), got))
	assert.Equal(t, int64(0), client.LLen(context.Background(), ProcessingKey).Val())
	assert.Equal(t, int64(0), client.ZCard(context.Background(), InFlightKey).Val())
}

func TestRedisQueueNackRequeues(t *testing.T) {
	rq, client := newRedisTestQueue(t)

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, rq.Submit(context.Background(), job))

	got, err := rq.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)

	assert.NoError(t, rq.Nack(context.Background(), got))

	// 重新入队：processing/in_flight 清空，queue 恢复 1 条
	assert.Equal(t, int64(0), client.LLen(context.Background(), ProcessingKey).Val())
	assert.Equal(t, int64(0), client.ZCard(context.Background(), InFlightKey).Val())
	assert.Equal(t, int64(1), client.LLen(context.Background(), QueueKey).Val())

	// 可再次消费
	got2, err := rq.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, got2.ID)
}

func TestRedisQueueRequeueExpired(t *testing.T) {
	rq, client := newRedisTestQueue(t)

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, rq.Submit(context.Background(), job))

	_, err := rq.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)

	// 未超时：RequeueExpired 返回 0
	n, err := rq.RequeueExpired(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	// 将 in_flight 的 score 改为过去时间戳，模拟 worker 崩溃后可见性超时
	res, err := client.ZRangeWithScores(context.Background(), InFlightKey, 0, -1).Result()
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	client.ZAdd(context.Background(), InFlightKey, redis.Z{
		Score:  float64(time.Now().Add(-time.Minute).Unix()),
		Member: res[0].Member,
	})

	n, err = rq.RequeueExpired(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	// 任务回到实时队列
	assert.Equal(t, int64(1), client.LLen(context.Background(), QueueKey).Val())
	assert.Equal(t, int64(0), client.ZCard(context.Background(), InFlightKey).Val())
	assert.Equal(t, int64(0), client.LLen(context.Background(), ProcessingKey).Val())
}
