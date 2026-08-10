// internal/platform/queue/rabbitmq_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQChannel 抽象 amqp.Channel 的消息收发能力，便于测试注入
type RabbitMQChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

// RabbitMQQueue RabbitMQ 消息队列实现
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel RabbitMQChannel
	queue   string
	msgs    <-chan amqp.Delivery // 构造时建立的一次性 consumer 订阅
}

// NewRabbitMQQueue 创建 RabbitMQ 队列
func NewRabbitMQQueue(cfg RabbitMQConfig) (*RabbitMQQueue, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	// 构造时建立一次 Consume 订阅并持有 delivery channel，
	// 避免 Consume 每次调用都新建 broker consumer 累积泄漏配额
	msgs, err := ch.Consume(cfg.QueueName, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return &RabbitMQQueue{conn: conn, channel: ch, queue: cfg.QueueName, msgs: msgs}, nil
}

// Submit 发布任务到队列
func (q *RabbitMQQueue) Submit(ctx context.Context, job *JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.channel.PublishWithContext(ctx, "", q.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         data,
		DeliveryMode: amqp.Persistent,
	})
}

// SubmitDelayed RabbitMQ 无原生延迟队列，当前直接发送，延迟由业务侧处理。
// 如需真实延迟，需引入延迟交换机 + TTL 或死信队列另行实现。
func (q *RabbitMQQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, delay+10*time.Second)
	defer cancel()
	return q.Submit(ctx, job)
}

// Consume 从队列取一条消息（从构造时建立的一次性订阅读取）
func (q *RabbitMQQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	select {
	case msg, ok := <-q.msgs:
		if !ok {
			return nil, fmt.Errorf("consumer channel closed")
		}
		var job JobData
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			return nil, fmt.Errorf("unmarshal rabbitmq message: %w", err)
		}
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("consume timeout")
	}
}

// Ack 确认任务。RabbitMQ autoAck=true，显式 ack 为 no-op。
func (q *RabbitMQQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack 否认任务。RabbitMQ at-most-once 语义下 autoAck=true，no-op；
// 重试由 WorkerPool 的持久化存储驱动。
func (q *RabbitMQQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// MoveDueJobs RabbitMQ 无延迟队列，返回 0 保持接口一致
func (q *RabbitMQQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
