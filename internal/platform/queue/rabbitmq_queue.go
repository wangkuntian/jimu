// internal/platform/queue/rabbitmq_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQChannel 抽象 amqp.Channel 的消息收发能力，便于测试注入
type RabbitMQChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

// RabbitMQQueue RabbitMQ 消息队列实现。
// 消费采用 at-least-once：autoAck=false，Consume 取到 delivery 后按 token 登记，
// 处理成功 Ack 确认、失败 Nack(requeue) 重新入队；worker 崩溃时连接关闭，
// 未确认的 delivery 由 broker 自动重投。
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel RabbitMQChannel
	queue   string
	msgs    <-chan amqp.Delivery // 构造时建立的一次性 consumer 订阅

	mu       sync.Mutex
	inFlight map[string]amqp.Delivery // token -> 未确认 delivery
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
	// autoAck=false：消费后需显式 Ack/Nack，配合可见性重投实现 at-least-once。
	// 构造时建立一次 Consume 订阅并持有 delivery channel，
	// 避免 Consume 每次调用都新建 broker consumer 累积泄漏配额。
	msgs, err := ch.Consume(cfg.QueueName, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return &RabbitMQQueue{
		conn:     conn,
		channel:  ch,
		queue:    cfg.QueueName,
		msgs:     msgs,
		inFlight: make(map[string]amqp.Delivery),
	}, nil
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

// Consume 从队列取一条消息（从构造时建立的一次性订阅读取），
// 生成 token 并登记未确认 delivery，供 Ack/Nack 精确匹配。
func (q *RabbitMQQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	select {
	case msg, ok := <-q.msgs:
		if !ok {
			return nil, fmt.Errorf("consumer channel closed")
		}
		var job JobData
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			// 毒消息：拒绝且不重入队（丢弃或进 DLQ），避免死循环
			_ = msg.Nack(false, false)
			return nil, fmt.Errorf("unmarshal rabbitmq message: %w", err)
		}
		job.Token = uuid.NewString()
		q.mu.Lock()
		if q.inFlight == nil {
			q.inFlight = make(map[string]amqp.Delivery)
		}
		q.inFlight[job.Token] = msg
		q.mu.Unlock()
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("consume timeout")
	}
}

// Ack 确认任务：向 broker 确认 delivery，任务完成。
func (q *RabbitMQQueue) Ack(ctx context.Context, job *JobData) error {
	q.mu.Lock()
	d, ok := q.inFlight[job.Token]
	if ok {
		delete(q.inFlight, job.Token)
	}
	q.mu.Unlock()
	if !ok {
		return nil // 未登记（已确认/已重投），幂等
	}
	return d.Ack(false)
}

// Nack 否认任务：requeue=true 将 delivery 重新入队供重试（at-least-once）。
func (q *RabbitMQQueue) Nack(ctx context.Context, job *JobData) error {
	q.mu.Lock()
	d, ok := q.inFlight[job.Token]
	if ok {
		delete(q.inFlight, job.Token)
	}
	q.mu.Unlock()
	if !ok {
		return nil
	}
	return d.Nack(false, true)
}

// MoveDueJobs RabbitMQ 无延迟队列，返回 0 保持接口一致
func (q *RabbitMQQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
