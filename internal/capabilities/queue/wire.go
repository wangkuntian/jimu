package queue

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配队列能力：返回作业与调度管理模块（/api/v1/admin/jobs*、/admin/tasks*）。
//
// 队列客户端**不在此构造**：base 行为是只在 outbox.publisher=mq 时于 outbox 的 MQ 分支
// 按 queue.type 构造（见 outbox.Wire），kafka/rabbitmq 的构造会连 broker 并 fail-fast，
// event_bus 下不得发生。queue 端口随之取消：唯一的消费方 outbox 直接 import 本能力构造。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// 驱动级可插拔（设计 §3.7）：本形态只编译了一部分队列驱动，配置里写了未编译的
	// 类型必须在启动时 fail-closed，而不是等到 outbox 走 MQ 时才暴露。
	if cfg := assembly.MustSection[*Config](ctx, ConfigKey); cfg != nil {
		if err := EnsureRegistered(cfg.Type); err != nil {
			return nil, err
		}
	}
	return NewModule(ctx.DB(), ctx.Scheduler()), nil
}
