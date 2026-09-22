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
	return NewModule(ctx.DB(), ctx.Scheduler()), nil
}
