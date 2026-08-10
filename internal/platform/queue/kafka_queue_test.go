// internal/platform/queue/kafka_queue_test.go
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*KafkaQueue)(nil)
	var _ Consumer = (*KafkaQueue)(nil)
}

// fakeKafkaWriter 内存假 writer，记录写入的消息
type fakeKafkaWriter struct {
	msgs []kafka.Message
	err  error
}

func (f *fakeKafkaWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, msgs...)
	return nil
}

// fakeKafkaReader 内存假 reader，按 FIFO 返回预先放入的消息
type fakeKafkaReader struct {
	msgs []kafka.Message
	err  error
}

func (f *fakeKafkaReader) ReadMessage(context.Context) (kafka.Message, error) {
	if f.err != nil {
		return kafka.Message{}, f.err
	}
	if len(f.msgs) == 0 {
		return kafka.Message{}, errors.New("no message")
	}
	msg := f.msgs[0]
	f.msgs = f.msgs[1:]
	return msg, nil
}

func TestKafkaQueue_SubmitConsume(t *testing.T) {
	w := &fakeKafkaWriter{}
	r := &fakeKafkaReader{}
	q := &KafkaQueue{writer: w, reader: r}

	job := &JobData{ID: 7, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	// Submit 应把任务 JSON 序列化写入 Kafka，key 为任务 ID
	require.Len(t, w.msgs, 1)
	assert.Equal(t, "7", string(w.msgs[0].Key))
	var got JobData
	assert.NoError(t, json.Unmarshal(w.msgs[0].Value, &got))
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Type, got.Type)
	assert.Equal(t, job.Payload, got.Payload)

	// 写入的消息可被 Consume 读回（submit → consume → ack 语义闭环）
	r.msgs = w.msgs
	consumed, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job, consumed)

	assert.NoError(t, q.Ack(context.Background(), consumed))
	assert.NoError(t, q.Nack(context.Background(), consumed))
}

func TestKafkaQueue_ConsumeUnmarshalError(t *testing.T) {
	r := &fakeKafkaReader{msgs: []kafka.Message{{Value: []byte("not-json")}}}
	q := &KafkaQueue{writer: &fakeKafkaWriter{}, reader: r}

	_, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.Error(t, err)
}
