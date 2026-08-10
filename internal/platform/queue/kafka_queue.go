// internal/platform/queue/kafka_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaQueue Kafka 消息队列实现
type KafkaQueue struct {
	writer *kafka.Writer
	reader *kafka.Reader
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

// SubmitDelayed Kafka 无原生延迟队列，使用消息头标注延迟，由消费端处理。
// 当前实现先直接发送（延迟由业务侧在 payload 中携带时间戳处理）。
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

// Ack Kafka 通过 offset 自动提交，显式 ack 无需额外操作
func (q *KafkaQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack Kafka 通过 offset 自动提交，显式 nack 无需额外操作
func (q *KafkaQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// MoveDueJobs Kafka 无延迟队列，返回 0 保持接口一致
func (q *KafkaQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
