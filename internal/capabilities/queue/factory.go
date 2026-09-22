package queue

import (
	"errors"
	"fmt"
	"sort"
	"strings"

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

// Factory 按配置构造队列实现。驱动包在 init() 中调用 Register 注册。
type Factory func(Config) (Queue, error)

// ErrUnregisteredDriver 标记「配置的类型未编译进本构建」，供调用方 errors.Is 判定。
var ErrUnregisteredDriver = errors.New("queue driver is not compiled into this build")

// drivers 是本构建已注册的驱动表（init 期写入，启动后只读，不加锁）。
var drivers = map[Type]Factory{}

// Register 注册队列驱动，仅供驱动包在 init() 中调用；重复注册 panic。
func Register(t Type, f Factory) {
	if _, dup := drivers[t]; dup {
		panic("queue: driver already registered: " + string(t))
	}
	drivers[t] = f
}

// RegisteredTypes 返回本构建已注册的队列类型（升序）。
func RegisteredTypes() []Type {
	out := make([]Type, 0, len(drivers))
	for t := range drivers {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// EnsureRegistered 校验配置的队列类型已编译进本构建（queue.Wire 在启动时调用）。
func EnsureRegistered(t Type) error {
	if _, ok := drivers[t]; !ok {
		return fmt.Errorf("queue driver %q is not compiled into this build (compiled: %s): %w",
			t, typesList(RegisteredTypes()), ErrUnregisteredDriver)
	}
	return nil
}

// New 按类型创建队列；类型未编译进本构建时明确报错，不静默回退。
func New(cfg Config) (Queue, error) {
	if err := EnsureRegistered(cfg.Type); err != nil {
		return nil, err
	}
	return drivers[cfg.Type](cfg)
}

// typesList 渲染已注册类型清单；空集渲染为 none。
func typesList(types []Type) string {
	if len(types) == 0 {
		return "none"
	}
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}
	return strings.Join(out, ", ")
}
