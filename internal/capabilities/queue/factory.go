package queue

import (
	"fmt"

	redistore "jimu/internal/kernel/redis"
)

// Type 队列类型
type Type string

const (
	TypeRedis    Type = "redis"
	TypeKafka    Type = "kafka"
	TypeRabbitMQ Type = "rabbitmq"
)

// Config 队列配置。Type/Kafka/RabbitMQ 来自 app.yaml 的 queue 段
// （mapstructure 标签保证与下沉前的键名映射一致）；Redis 为装配期注入，不来自配置。
type Config struct {
	Type     Type             `mapstructure:"type"`
	Redis    redistore.Client `mapstructure:"-"`
	Kafka    KafkaConfig      `mapstructure:"kafka"`
	RabbitMQ RabbitMQConfig   `mapstructure:"rabbitmq"`
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
