// internal/platform/queue/kafka_config.go
package queue

// KafkaConfig Kafka 队列配置
type KafkaConfig struct {
	Brokers  []string // broker 地址列表
	Topic    string   // 主主题
	GroupID  string   // 消费组 ID
	MaxRetry int      // 重试次数
}
