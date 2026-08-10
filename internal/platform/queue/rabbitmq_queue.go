// internal/platform/queue/rabbitmq_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQQueue RabbitMQ 消息队列实现
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
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
	return &RabbitMQQueue{conn: conn, channel: ch, queue: cfg.QueueName}, nil
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

// SubmitDelayed RabbitMQ 用延迟交换机或 TTL 实现。当前直接发送，延迟由消费端处理。
func (q *RabbitMQQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, delay+10*time.Second)
	defer cancel()
	return q.Submit(ctx, job)
}

// Consume 从队列取一条消息
func (q *RabbitMQQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	msgs, err := q.channel.Consume(q.queue, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	select {
	case msg, ok := <-msgs:
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

// Ack RabbitMQ autoAck=true，显式 ack 无需额外操作
func (q *RabbitMQQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack RabbitMQ autoAck=true，显式 nack 无需额外操作
func (q *RabbitMQQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// MoveDueJobs RabbitMQ 无延迟队列，返回 0 保持接口一致
func (q *RabbitMQQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
