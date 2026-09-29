package queue

import (
	"fmt"

	"jimu/internal/kernel/scheduler"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "queue"

// SchedulerConfigKey 调度器配置段键。调度器实例由本能力用于作业调度
// （/admin/tasks*、job_history），其配置随之归本能力。
const SchedulerConfigKey = scheduler.ConfigKey

// 调度器存储类型取值（类型与实现归位内核 scheduler 包，此处保留既有引用）。
const (
	SchedulerStoreMemory = scheduler.StoreMemory
	SchedulerStoreMySQL  = scheduler.StoreMySQL
)

// SchedulerConfig 调度器配置段（原 queue 自有类型，P2.4 归位内核 scheduler 包）。
type SchedulerConfig = scheduler.Config

var (
	validQueueTypes = []Type{TypeRedis, TypeKafka, TypeRabbitMQ}
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
