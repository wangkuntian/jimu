package queue

import (
	"fmt"

	redistore "jimu/internal/platform/redis"
)

// Type 队列类型
type Type string

const (
	TypeRedis    Type = "redis"
	TypeKafka    Type = "kafka"
	TypeRabbitMQ Type = "rabbitmq"
)

// Config 队列配置
type Config struct {
	Type     Type
	Redis    redistore.Client
	Kafka    KafkaConfig
	RabbitMQ RabbitMQConfig
}

// New 按类型创建队列
func New(cfg Config) (Queue, error) {
	switch cfg.Type {
	case TypeRedis:
		return NewRedisQueue(cfg.Redis), nil
	case TypeKafka:
		return NewKafkaQueue(cfg.Kafka)
	case TypeRabbitMQ:
		return NewRabbitMQQueue(cfg.RabbitMQ)
	default:
		return nil, fmt.Errorf("invalid queue type: %q", cfg.Type)
	}
}
