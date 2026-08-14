// internal/platform/queue/kafka_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaMessageWriter 抽象 kafka.Writer 写消息能力，便于测试注入
type KafkaMessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaMessageReader 抽象 kafka.Reader 读消息能力，便于测试注入
type KafkaMessageReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
}

// KafkaQueue Kafka 消息队列实现
type KafkaQueue struct {
	writer KafkaMessageWriter
	reader KafkaMessageReader
}

// NewKafkaQueue 创建 Kafka 队列
func NewKafkaQueue(cfg KafkaConfig) (*KafkaQueue, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" {
		return nil, fmt.Errorf("kafka brokers and topic required")
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &KafkaQueue{writer: w, reader: r}, nil
}

// Submit 提交任务到 Kafka topic
func (q *KafkaQueue) Submit(ctx context.Context, job *JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", job.ID)),
		Value: data,
	})
}

// SubmitDelayed Kafka 无原生延迟队列，当前直接发送，延迟由业务侧处理。
// 如需真实延迟，需引入定时中间件（如延迟 topic + 调度器）另行实现。
func (q *KafkaQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, delay+10*time.Second)
	defer cancel()
	return q.Submit(ctx, job)
}

// Consume 从 Kafka 读取一条消息
func (q *KafkaQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	msgCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	msg, err := q.reader.ReadMessage(msgCtx)
	if err != nil {
		return nil, err
	}
	var job JobData
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		return nil, fmt.Errorf("unmarshal kafka message: %w", err)
	}
	return &job, nil
}

// Ack 确认任务。Kafka at-most-once 语义下 offset 自动提交，显式 ack 为 no-op。
func (q *KafkaQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack 否认任务：重新发布到 topic 触发重试（应用层 at-least-once）。
// Kafka 的 offset 在 Consume（ReadMessage）时已自动提交，崩溃窗口内消息可能丢失；
// 但处理失败的任务经此重投，配合 WorkerPool 的持久化存储保证重试直至耗尽 MaxAttempts。
func (q *KafkaQueue) Nack(ctx context.Context, job *JobData) error {
	return q.Submit(ctx, job)
}

// MoveDueJobs Kafka 无延迟队列，返回 0 保持接口一致
func (q *KafkaQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
