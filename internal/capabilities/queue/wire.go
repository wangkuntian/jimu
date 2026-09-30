package queue

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配队列能力：返回作业与调度管理模块（/api/v1/admin/jobs*、/admin/tasks*）。
//
// 队列客户端不在此构造；惰性工厂只在 outbox.publisher=mq 时创建驱动和 worker。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// 驱动级可插拔（设计 §3.7）：本形态只编译了一部分队列驱动，配置里写了未编译的
	// 类型必须在启动时 fail-closed，而不是等到 outbox 走 MQ 时才暴露。
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg != nil {
		if err := EnsureRegistered(cfg.Type); err != nil {
			return nil, err
		}
	}
	factory := &outboxMQFactory{db: ctx.DB(), newQueue: New, registerComponent: ctx.RegisterComponent}
	if cfg != nil {
		factory.cfg = *cfg
		factory.cfg.Redis = ctx.Redis()
	}
	if err := ctx.Provide(contract.OutboxMQPortName, factory); err != nil {
		return nil, err
	}
	return NewModule(ctx.DB(), ctx.Scheduler()), nil
}
