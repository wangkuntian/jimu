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
type Consumer interface {
	// Consume 消费任务（阻塞式，timeout 内无任务返回错误）
	Consume(ctx context.Context, timeout time.Duration) (*JobData, error)
	// Ack 确认任务处理成功
	Ack(ctx context.Context, job *JobData) error
	// Nack 否认任务处理。
	// Redis：重新入队（BLMove 原子消费 + 可见性超时兜底，at-least-once）；
	// RabbitMQ：autoAck=false + requeue 重新入队（at-least-once，连接断开时 broker 自动重投未确认消息）；
	// Kafka：offset 自动提交，Nack 重新发布到 topic 触发重试（应用层 at-least-once，
	// 崩溃窗口内消息可能丢失），重试上限由 WorkerPool 的持久化存储（MySQL store）驱动。
	Nack(ctx context.Context, job *JobData) error
}
