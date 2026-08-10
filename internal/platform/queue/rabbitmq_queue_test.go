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

func TestRabbitMQQueue_SubmitConsume(t *testing.T) {
	ch := &fakeRabbitMQChannel{deliveries: make(chan amqp.Delivery, 1)}
	q := &RabbitMQQueue{channel: ch, queue: "test-queue", msgs: ch.deliveries}

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
	ch.deliveries <- amqp.Delivery{Body: ch.published[0].Body}
	consumed, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job, consumed)

	assert.NoError(t, q.Ack(context.Background(), consumed))
	assert.NoError(t, q.Nack(context.Background(), consumed))
}

func TestRabbitMQQueue_ConsumeUnmarshalError(t *testing.T) {
	ch := &fakeRabbitMQChannel{deliveries: make(chan amqp.Delivery, 1)}
	ch.deliveries <- amqp.Delivery{Body: []byte("not-json")}
	q := &RabbitMQQueue{channel: ch, queue: "test-queue", msgs: ch.deliveries}

	_, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.Error(t, err)
}

func TestRabbitMQQueue_ConsumeTimeout(t *testing.T) {
	ch := &fakeRabbitMQChannel{deliveries: make(chan amqp.Delivery)}
	q := &RabbitMQQueue{channel: ch, queue: "test-queue", msgs: ch.deliveries}

	_, err := q.Consume(context.Background(), 50*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
