package queue

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配队列能力：按 queue.type 构造队列实例并暴露为端口（outbox 的 MQ 发布器经它
// 消费），返回作业与调度管理模块（/api/v1/admin/jobs*、/admin/tasks*）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	queueCfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if queueCfg == nil {
		queueCfg = &Config{}
	}
	cfg := *queueCfg
	cfg.Redis = ctx.Redis()
	q, err := New(cfg)
	if err != nil {
		return nil, fmt.Errorf("init queue: %w", err)
	}
	if err := ctx.Provide(PortName, q); err != nil {
		return nil, fmt.Errorf("provide queue port: %w", err)
	}
	return NewModule(ctx.DB(), ctx.Scheduler()), nil
}
