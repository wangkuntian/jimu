package queue

import (
	"fmt"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "queue"

// SchedulerConfigKey 调度器配置段键。调度器实例由本能力用于作业调度
// （/admin/tasks*、job_history），其配置随之归本能力。
const SchedulerConfigKey = "scheduler"

// 调度器存储类型取值。
const (
	SchedulerStoreMemory = "memory"
	SchedulerStoreMySQL  = "mysql"
)

var (
	validQueueTypes      = []Type{TypeRedis, TypeKafka, TypeRabbitMQ}
	validSchedulerStores = []string{SchedulerStoreMemory, SchedulerStoreMySQL}
	// mqQueueTypes outbox 走 MQ 时允许的队列类型。
	mqQueueTypes = []Type{TypeKafka, TypeRabbitMQ, TypeRedis}
)

// ApplyDefaults 本能力无配置层默认值：队列/调度器类型必填，由 Validate 兜底。
func (c *Config) ApplyDefaults() {}

// Validate 校验队列配置段。仅在本能力启用时由组合根调用（设计 §8）。
func (c *Config) Validate() error {
	for _, t := range validQueueTypes {
		if c.Type == t {
			return nil
		}
	}
	return fmt.Errorf("queue.type: %q, must be one of %v", c.Type, validQueueTypes)
}

// SchedulerConfig 调度器配置段（原 config.SchedulerConfig，P2.1 下沉）。
type SchedulerConfig struct {
	Store string `mapstructure:"store"` // 任务定义存储类型：memory, mysql
}

// ApplyDefaults 本配置段无配置层默认值（container 对非 mysql 一律按 memory 处理）。
func (c *SchedulerConfig) ApplyDefaults() {}

// Validate 校验调度器配置段。
func (c *SchedulerConfig) Validate() error {
	for _, s := range validSchedulerStores {
		if c.Store == s {
			return nil
		}
	}
	return fmt.Errorf("scheduler.store: %q, must be one of %v", c.Store, validSchedulerStores)
}

// Load 解码并校验队列配置段。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadScheduler 解码并校验调度器配置段。
func LoadScheduler(dec config.SectionDecoder) (*SchedulerConfig, error) {
	var c SchedulerConfig
	if err := config.LoadSection(dec, SchedulerConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// SupportsOutboxMQ 报告该队列类型能否承载 outbox 的 MQ 投递。
// 供组合根做跨能力校验（outbox.publisher=mq 依赖 queue.type）。
func SupportsOutboxMQ(queueType Type) bool {
	for _, t := range mqQueueTypes {
		if queueType == t {
			return true
		}
	}
	return false
}
