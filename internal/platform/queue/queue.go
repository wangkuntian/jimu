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

// Consumer 消费者接口。
// Redis 为破坏性消费（BRPop），Ack/Nack 为 no-op。
type Consumer interface {
	// Consume 消费任务（阻塞式，timeout 内无任务返回错误）
	Consume(ctx context.Context, timeout time.Duration) (*JobData, error)
	// Ack 确认任务处理成功
	Ack(ctx context.Context, job *JobData) error
	// Nack 否认任务处理。
	// Redis/Kafka/RabbitMQ 当前均为 at-most-once 语义，Nack 为 no-op；
	// 重试由 WorkerPool 的持久化存储（MySQL store）驱动，不依赖队列 Nack。
	Nack(ctx context.Context, job *JobData) error
}
