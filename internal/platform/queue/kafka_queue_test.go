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

// fakeKafkaReader 内存假 reader，按 FIFO 返回预先放入的消息。
// FetchMessage 不自动提交，CommitMessages 记录已提交 offset。
type fakeKafkaReader struct {
	msgs    []kafka.Message
	err     error
	commits []kafka.Message // 已提交 offset 的消息
}

func (f *fakeKafkaReader) FetchMessage(context.Context) (kafka.Message, error) {
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

func (f *fakeKafkaReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	f.commits = append(f.commits, msgs...)
	return nil
}

func TestKafkaQueue_SubmitConsume(t *testing.T) {
	w := &fakeKafkaWriter{}
	r := &fakeKafkaReader{}
	q := &KafkaQueue{writer: w, reader: r, inFlight: make(map[uint64]kafka.Message)}

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

	// Ack 提交 offset，完成 at-least-once 闭环
	assert.NoError(t, q.Ack(context.Background(), consumed))
	require.Len(t, r.commits, 1)
	assert.Equal(t, uint64(7), consumed.ID)

	// Ack 幂等：重复 Ack 不再提交
	assert.NoError(t, q.Ack(context.Background(), consumed))
	assert.Len(t, r.commits, 1)
}

// TestKafkaQueue_NackDoesNotCommit 验证 Nack 不提交 offset，
// broker 重启后会重新投递未提交区间（at-least-once）。
func TestKafkaQueue_NackDoesNotCommit(t *testing.T) {
	w := &fakeKafkaWriter{}
	r := &fakeKafkaReader{msgs: []kafka.Message{{Key: []byte("1"), Value: []byte(`{"id":1,"type":"t","payload":""}`)}}}
	q := &KafkaQueue{writer: w, reader: r, inFlight: make(map[uint64]kafka.Message)}

	consumed, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), consumed.ID)

	assert.NoError(t, q.Nack(context.Background(), consumed))
	assert.Empty(t, r.commits, "Nack 不应提交 offset")
}

// TestKafkaQueue_ConsumeUnmarshalErrorCommits 验证 poison message 提交跳过，避免消费死循环。
func TestKafkaQueue_ConsumeUnmarshalErrorCommits(t *testing.T) {
	r := &fakeKafkaReader{msgs: []kafka.Message{{Value: []byte("not-json")}}}
	q := &KafkaQueue{writer: &fakeKafkaWriter{}, reader: r, inFlight: make(map[uint64]kafka.Message)}

	_, err := q.Consume(context.Background(), 100*time.Millisecond)
	assert.Error(t, err)
	require.Len(t, r.commits, 1, "poison message 应提交 offset 跳过")
}
