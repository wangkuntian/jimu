package queue

import (
	"fmt"

	"github.com/redis/go-redis/v9"
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
	Redis    *redis.Client
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
