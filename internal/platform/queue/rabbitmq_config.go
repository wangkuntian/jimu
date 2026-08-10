// internal/platform/queue/rabbitmq_config.go
package queue

// RabbitMQConfig RabbitMQ 队列配置
type RabbitMQConfig struct {
	URL       string // AMQP URL
	QueueName string // 队列名
	Exchange  string // 交换机名
}
