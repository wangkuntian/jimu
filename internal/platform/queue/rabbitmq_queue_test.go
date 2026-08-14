// internal/platform/queue/rabbitmq_queue_test.go
package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRabbitMQQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RabbitMQQueue)(nil)
	var _ Consumer = (*RabbitMQQueue)(nil)
}

// fakeRabbitMQChannel 内存假 channel，记录发布的消息并返回预设 delivery
type fakeRabbitMQChannel struct {
	published  []amqp.Publishing
	deliveries chan amqp.Delivery
}

func (f *fakeRabbitMQChannel) PublishWithContext(_ context.Context, _ string, _ string, _ bool, _ bool, msg amqp.Publishing) error {
	f.published = append(f.published, msg)
	return nil
}

func (f *fakeRabbitMQChannel) Consume(_ string, _ string, _ bool, _ bool, _ bool, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	return f.deliveries, nil
}

// fakeAcknowledger 记录 Ack/Nack/Reject 调用，模拟 broker 确认行为
type fakeAcknowledger struct {
	acks    []uint64
	nacks   []uint64
	rejects []uint64
}

func (f *fakeAcknowledger) Ack(tag uint64, _ bool) error {
	f.acks = append(f.acks, tag)
	return nil
}

func (f *fakeAcknowledger) Nack(tag uint64, _ bool, _ bool) error {
	f.nacks = append(f.nacks, tag)
	return nil
}

func (f *fakeAcknowledger) Reject(tag uint64, _ bool) error {
	f.rejects = append(f.rejects, tag)
	return nil
}

func newTestRabbitQueue(t *testing.T) (*RabbitMQQueue, *fakeRabbitMQChannel) {
	t.Helper()
	ch := &fakeRabbitMQChannel{deliveries: make(chan amqp.Delivery, 2)}
	q := &RabbitMQQueue{
		channel:  ch,
		queue:    "test-queue",
		msgs:     ch.deliveries,
		inFlight: make(map[string]amqp.Delivery),
	}
	return q, ch
}

func TestRabbitMQQueue_SubmitConsumeAck(t *testing.T) {
	q, ch := newTestRabbitQueue(t)

	job := &JobData{ID: 9, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	// Submit 应把任务 JSON 序列化持久化发布到队列
	require.Len(t, ch.published, 1)
	assert.Equal(t, "application/json", ch.published[0].ContentType)
	assert.Equal(t, amqp.Persistent, ch.published[0].DeliveryMode)
	var got JobData
	assert.NoError(t, json.Unmarshal(ch.published[0].Body, &got))
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Type, got.Type)
	assert.Equal(t, job.Payload, got.Payload)

	// 发布的消息可被 Consume 读回（submit → consume → ack 语义闭环）
	ack := &fakeAcknowledger{}
	ch.deliveries <- amqp.Delivery{Body: ch.published[0].Body, DeliveryTag: 1, Acknowledger: ack}
	consumed, err := q.Consume(context.Background(), 100*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, job.ID, consumed.ID)
	assert.Equal(t, job.Type, consumed.Type)
	assert.Equal(t, job.Payload, consumed.Payload)
	assert.NotEmpty(t, consumed.Token, "Consume 应生成唯一 token")

	assert.NoError(t, q.Ack(context.Background(), consumed))
	assert.Equal(t, []uint64{1}, ack.acks, "Ack 应确认对应 delivery")
}

func TestRabbitMQQueue_NackRequeues(t *testing.T) {
	q, ch := newTestRabbitQueue(t)

	job := &JobData{ID: 9, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	ack := &fakeAcknowledger{}
	ch.deliveries <- amqp.Delivery{Body: ch.published[0].Body, DeliveryTag: 2, Acknowledger: ack}
	consumed, err := q.Consume(context.Background(), 100*time.Millisecond)
	require.NoError(t, err)

	assert.NoError(t, q.Nack(context.Background(), consumed))
	assert.Equal(t, []uint64{2}, ack.nacks, "Nack 应 requeue 对应 delivery")

	// Nack 后 token 已从 inFlight 移除，重复 Nack 幂等且不再触碰 broker
	assert.NoError(t, q.Nack(context.Background(), consumed))
	assert.Len(t, ack.nacks, 1)
}

func TestRabbitMQQueue_ConsumeUnmarshalError(t *testing.T) {
	q, ch := newTestRabbitQueue(t)

	ack := &fakeAcknowledger{}
	ch.deliveries <- amqp.Delivery{Body: []byte("not-json"), DeliveryTag: 3, Acknowledger: ack}

	_, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.Error(t, err)
	assert.Len(t, ack.nacks, 1, "毒消息应被 Nack（拒绝且不重入队）")
}

func TestRabbitMQQueue_ConsumeTimeout(t *testing.T) {
	q, _ := newTestRabbitQueue(t)

	_, err := q.Consume(context.Background(), 50*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
