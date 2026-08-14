package outbox

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"jimu/internal/platform/queue"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// e2eStore 返回固定待发布事件，并捕获已发布 ID。
type e2eStore struct {
	events []Event
	marked []uint64
}

func (s *e2eStore) Add(ctx context.Context, tx interface{}, events ...Event) error { return nil }
func (s *e2eStore) FetchUnpublish(ctx context.Context, limit int) ([]Event, error) {
	return s.events, nil
}
func (s *e2eStore) MarkPublished(ctx context.Context, ids []uint64) error {
	s.marked = append(s.marked, ids...)
	return nil
}
func (s *e2eStore) MarkFailed(ctx context.Context, id uint64, err error) error { return nil }

// TestOutboxQueueWorkerEndToEnd 验证 outbox → MQ → worker 全链路传输闭环：
// Process 取事件 → MQPublisher 提交到 RedisQueue → WorkerPool 消费 → 处理器执行 → Ack。
// 全内存（miniredis），不依赖外部 Redis/MQ/DB。
func TestOutboxQueueWorkerEndToEnd(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	rq := queue.NewRedisQueue(client)

	// 记录处理器收到的载荷，验证跨队列序列化完整
	var handled atomic.Int32
	payloadCh := make(chan string, 1)
	queue.RegisterWorker("outbox:e2e.UserCreated", func(ctx context.Context, payload string) error {
		handled.Add(1)
		payloadCh <- payload
		return nil
	})

	store := &e2eStore{events: []Event{{
		ID:          1,
		AggregateID: "user-1",
		EventType:   "e2e.UserCreated",
		Payload:     json.RawMessage(`{"name":"tom"}`),
	}}}
	o := New(store, NewMQPublisher(rq))

	// worker 以 RedisQueue 为消费者；store 传 nil，outbox 事件无 jobs 行，跳过状态机
	wp := queue.NewWorkerPool(queue.WorkerConfig{
		Workers:     1,
		QueueName:   queue.QueueKey,
		PollTimeout: 1 * time.Second,
		MaxRetries:  3,
	}, rq, nil)
	wp.Start()
	t.Cleanup(wp.Stop)

	published, err := o.Process(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, published, "应发布 1 个事件")

	require.Eventually(t, func() bool {
		return handled.Load() == 1
	}, 2*time.Second, 10*time.Millisecond, "worker 应消费并处理事件")

	// 载荷完整：EventPayload 反序列化后字段保留
	var ev EventPayload
	require.NoError(t, json.Unmarshal([]byte(<-payloadCh), &ev))
	require.Equal(t, uint64(1), ev.ID)
	require.Equal(t, "user-1", ev.AggregateID)
	require.JSONEq(t, `{"name":"tom"}`, string(ev.Payload))

	require.Equal(t, []uint64{1}, store.marked, "事件应被标记为已发布")
}
