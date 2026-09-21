// internal/capabilities/queue/rabbitmq_config.go
package queue

// RabbitMQConfig RabbitMQ 队列配置
type RabbitMQConfig struct {
	URL       string `mapstructure:"url"`      // AMQP URL
	QueueName string `mapstructure:"queue"`    // 队列名（YAML 键沿用 queue）
	Exchange  string `mapstructure:"exchange"` // 交换机名
}
