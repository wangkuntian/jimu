package queue

import (
	"context"
	"time"
)

// Queue 生产者接口，所有队列实现（Redis/Kafka/RabbitMQ）都必须支持
type Queue interface {
	// Submit 提交任务到实时队列
	Submit(ctx context.Context, job *JobData) error
	// SubmitDelayed 提交延迟任务
	SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error
	// MoveDueJobs 将到期的延迟任务移入实时队列
	MoveDueJobs(ctx context.Context) (int, error)
}

// Consumer 消费者接口。Kafka/RabbitMQ 天然支持 ack 语义；
// Redis 为破坏性消费（BRPop），Ack/Nack 实现为 no-op。
type Consumer interface {
	// Consume 消费任务（阻塞式，timeout 内无任务返回错误）
	Consume(ctx context.Context, timeout time.Duration) (*JobData, error)
	// Ack 确认任务处理成功
	Ack(ctx context.Context, job *JobData) error
	// Nack 否认任务处理，可触发重试
	Nack(ctx context.Context, job *JobData) error
}
