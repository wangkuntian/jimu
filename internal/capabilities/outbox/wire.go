package outbox

import (
	"context"
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
)

// Wire 装配 outbox 能力：构造事件存储与发布器，把 *Outbox 暴露为端口供 user/auth 写入
// 事件，并注册 outbox_process 定时任务。
//
// publisher=event_bus（默认）：订阅事件总线 outbox:* 主题桥接到裸业务主题，不构造队列；
// publisher=mq：通过 queue 提供的惰性工厂构造客户端与 worker，配置或 broker 错误
// 在装配期失败。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	outboxStore := NewMySQLStore(ctx.DB())
	var factory contract.OutboxMQFactory
	if cfg.UsesMQ() {
		factory, _ = ctx.Port(contract.OutboxMQPortName).(contract.OutboxMQFactory)
	}
	publisher, err := selectPublisher(context.Background(), *cfg, factory, ctx.EventBus(), ctx.Logger())
	if err != nil {
		return nil, err
	}
	processor := New(outboxStore, publisher)
	if err := ctx.Provide(PortName, processor); err != nil {
		return nil, fmt.Errorf("provide outbox port: %w", err)
	}
	if err := ctx.RegisterJob(scheduler.Job{ID: "outbox_process", Name: "Process Outbox Events", Spec: "@every 10s", Run: func() {
		n, err := processor.Process(context.Background(), 100)
		if err != nil {
			ctx.Logger().Errorw("outbox process error", "error", err.Error())
		} else if n > 0 {
			ctx.Logger().Debugw("outbox processed", "count", n)
		}
	}}); err != nil {
		return nil, err
	}
	if job, ok := newOutboxRetentionJob(ctx.DB(), cfg.Retention, ctx.Logger()); ok {
		if err := ctx.RegisterJob(job); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func selectPublisher(ctx context.Context, cfg Config, factory contract.OutboxMQFactory, bus *event.EventBus, log *logger.Logger) (Publisher, error) {
	if !cfg.UsesMQ() {
		RegisterEventBusBridge(bus, log)
		return NewEventBusPublisher(bus), nil
	}
	if factory == nil {
		return nil, fmt.Errorf("outbox MQ port %q unavailable", contract.OutboxMQPortName)
	}
	types := make([]string, 0, len(eventTypeConverters))
	for eventType := range eventTypeConverters {
		types = append(types, eventType)
	}
	bridge := BridgeWorker(bus)
	publisher, err := factory.StartOutbox(ctx, types, bridge)
	if err != nil {
		return nil, fmt.Errorf("init outbox queue: %w", err)
	}
	return NewMQPublisher(publisher), nil
}
