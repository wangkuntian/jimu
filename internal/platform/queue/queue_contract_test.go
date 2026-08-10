// internal/platform/queue/queue_contract_test.go
package queue

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// newRedisTestQueue 创建指向本地 Redis（DB 15）的测试队列；Redis 不可用时跳过测试
func newRedisTestQueue(t *testing.T) *RedisQueue {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15, // 使用 DB 15 避免污染其他数据
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// 清理测试数据
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("FlushDB failed: %v", err)
	}

	return NewRedisQueue(client)
}

func TestRedisQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RedisQueue)(nil)
	var _ Consumer = (*RedisQueue)(nil)
}

func TestQueueContract_SubmitConsume(t *testing.T) {
	var q Queue = newRedisTestQueue(t)
	var c Consumer = q.(Consumer)

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	got, err := c.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Payload, got.Payload)

	// Redis 为破坏性消费，Ack/Nack 为 no-op
	assert.NoError(t, c.Ack(context.Background(), got))
	assert.NoError(t, c.Nack(context.Background(), got))
}
