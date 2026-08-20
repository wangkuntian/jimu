// internal/platform/queue/kafka_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaMessageWriter 抽象 kafka.Writer 写消息能力，便于测试注入
type KafkaMessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaMessageReader 抽象 kafka.Reader 读消息能力，便于测试注入。
// FetchMessage 不提交 offset，CommitMessages 显式提交——at-least-once 语义基础。
type KafkaMessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaQueue Kafka 消息队列实现。
// 消费采用 at-least-once：Consume 用 FetchMessage 取消息（不自动提交 offset），
// 处理成功 Ack 显式 CommitMessages 提交 offset；失败 Nack 不提交，
// worker 崩溃重启后未提交区间由 broker 重新投递。poison message 提交跳过避免死循环。
type KafkaQueue struct {
	writer KafkaMessageWriter
	reader KafkaMessageReader

	mu       sync.Mutex
	inFlight map[uint64]kafka.Message // jobID -> 未提交 offset 的消息
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
	return &KafkaQueue{
		writer:   w,
		reader:   r,
		inFlight: make(map[uint64]kafka.Message),
	}, nil
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

// Consume 从 Kafka 拉取一条消息（不提交 offset），登记 inFlight 供 Ack 精确匹配。
func (q *KafkaQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	msgCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	msg, err := q.reader.FetchMessage(msgCtx)
	if err != nil {
		return nil, err
	}
	var job JobData
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		// poison message：提交该 offset 跳过，避免消费死循环卡住分区
		_ = q.reader.CommitMessages(ctx, msg)
		return nil, fmt.Errorf("unmarshal kafka message: %w", err)
	}
	q.mu.Lock()
	q.inFlight[job.ID] = msg
	q.mu.Unlock()
	return &job, nil
}

// Ack 确认任务：提交该消息 offset，任务完成（at-least-once）。
func (q *KafkaQueue) Ack(ctx context.Context, job *JobData) error {
	q.mu.Lock()
	msg, ok := q.inFlight[job.ID]
	if ok {
		delete(q.inFlight, job.ID)
	}
	q.mu.Unlock()
	if !ok {
		return nil // 未登记（已提交/已重投），幂等
	}
	return q.reader.CommitMessages(ctx, msg)
}

// Nack 否认任务：不提交 offset，broker 在下次 poll 或崩溃重启后重新投递未提交区间。
// （应用层 at-least-once；消息可能重复投递，消费端须幂等。）
func (q *KafkaQueue) Nack(ctx context.Context, job *JobData) error {
	q.mu.Lock()
	delete(q.inFlight, job.ID)
	q.mu.Unlock()
	return nil
}

// MoveDueJobs Kafka 无延迟队列，返回 0 保持接口一致
func (q *KafkaQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
