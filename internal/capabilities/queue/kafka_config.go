// internal/capabilities/queue/kafka_config.go
package queue

// KafkaConfig Kafka 队列配置
type KafkaConfig struct {
	Brokers  []string `mapstructure:"brokers"`   // broker 地址列表
	Topic    string   `mapstructure:"topic"`     // 主主题
	GroupID  string   `mapstructure:"group_id"`  // 消费组 ID
	MaxRetry int      `mapstructure:"max_retry"` // 重试次数
}
