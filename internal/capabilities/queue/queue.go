package queue

import (
	"context"
	"time"
)

// Redis 驱动的队列键名（RedisQueue 与 WorkerConfig 默认队列名共用；键名与下沉前各值
// 逐字节一致，不随驱动下沉改名）。可见性超时 visibilityTTL 属驱动内部，随驱动包下沉。
const (
	QueueKey      = "jimu:queue:default"
	DelayedKey    = "jimu:queue:delayed"
	ProcessingKey = "jimu:queue:processing" // 处理中任务列表（已消费未确认）
	InFlightKey   = "jimu:queue:in_flight"  // 处理中任务 ZSET：member=任务 JSON，score=可见性超时时间戳
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
	// Kafka：FetchMessage 不提交 offset，Nack 不提交 → broker 重新投递未提交区间（at-least-once），
	// 消费端须幂等。重试上限由 WorkerPool 的持久化存储（MySQL store）驱动。
	Nack(ctx context.Context, job *JobData) error
}

// JobData 队列中的任务数据（各驱动共用：序列化后写入队列，消费侧反序列化）。
// Token/Deadline 由消费者实现填写，用于可见性超时与 Ack/Nack 精确匹配。
type JobData struct {
	ID          uint64 `json:"id"`
	Type        string `json:"type"`
	Payload     string `json:"payload"`
	Token       string `json:"token,omitempty"`       // 单次消费唯一标识，区分重复入队的同名任务
	Deadline    int64  `json:"deadline,omitempty"`    // 可见性超时时间戳（unix 秒）
	Traceparent string `json:"traceparent,omitempty"` // W3C 追踪上下文，跨 MQ 透传
	Tracestate  string `json:"tracestate,omitempty"`  // W3C 追踪状态，跨 MQ 透传
}
