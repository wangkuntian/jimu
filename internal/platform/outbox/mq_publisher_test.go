package outbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/platform/queue"
)

// fakeQueue 内存假队列，验证 MQPublisher 发布语义
type fakeQueue struct {
	submitted []*queue.JobData
}

func (f *fakeQueue) Submit(ctx context.Context, job *queue.JobData) error {
	f.submitted = append(f.submitted, job)
	return nil
}

func (f *fakeQueue) SubmitDelayed(ctx context.Context, job *queue.JobData, delay time.Duration) error {
	return nil
}

func (f *fakeQueue) MoveDueJobs(ctx context.Context) (int, error) { return 0, nil }

// TestMQPublisher_Publish 验证事件序列化后按 outbox: 前缀提交到队列
func TestMQPublisher_Publish(t *testing.T) {
	fq := &fakeQueue{}
	p := NewMQPublisher(fq)

	events := []Event{
		{
			ID:          1,
			AggregateID: "user-1",
			EventType:   "UserCreated",
			Payload:     json.RawMessage(`{"name":"tom"}`),
			Metadata:    json.RawMessage(`{"trace":"abc"}`),
			RetryCount:  0,
		},
		{
			ID:          2,
			AggregateID: "order-1",
			EventType:   "OrderPlaced",
			Payload:     json.RawMessage(`{"amount":100}`),
		},
	}

	err := p.Publish(context.Background(), events...)
	require.NoError(t, err)
	require.Len(t, fq.submitted, 2)

	// job Type 带 outbox: 前缀，ID 复用事件 ID，Payload 为事件 JSON 序列化结果
	assert.Equal(t, uint64(1), fq.submitted[0].ID)
	assert.Equal(t, "outbox:UserCreated", fq.submitted[0].Type)
	assert.JSONEq(t, `{"id":1,"aggregate_id":"user-1","event_type":"UserCreated","payload":{"name":"tom"},"metadata":{"trace":"abc"},"created_at":"0001-01-01T00:00:00Z","retry_count":0}`, fq.submitted[0].Payload)

	assert.Equal(t, uint64(2), fq.submitted[1].ID)
	assert.Equal(t, "outbox:OrderPlaced", fq.submitted[1].Type)
	assert.JSONEq(t, `{"id":2,"aggregate_id":"order-1","event_type":"OrderPlaced","payload":{"amount":100},"created_at":"0001-01-01T00:00:00Z","retry_count":0}`, fq.submitted[1].Payload)
}

// TestMQPublisher_PublishError 验证队列提交失败时返回错误并停止后续发布
func TestMQPublisher_PublishError(t *testing.T) {
	// 自定义错误队列，Submit 失败
	ep := NewMQPublisher(&errorQueue{})

	events := []Event{
		{ID: 1, EventType: "UserCreated", Payload: json.RawMessage(`{"name":"tom"}`)},
	}
	err := ep.Publish(context.Background(), events...)
	require.Error(t, err)
	assert.ErrorContains(t, err, "submit outbox event 1")
}

// errorQueue Submit 始终返回错误
type errorQueue struct{}

func (e *errorQueue) Submit(ctx context.Context, job *queue.JobData) error {
	return assert.AnError
}

func (e *errorQueue) SubmitDelayed(ctx context.Context, job *queue.JobData, delay time.Duration) error {
	return nil
}

func (e *errorQueue) MoveDueJobs(ctx context.Context) (int, error) { return 0, nil }
