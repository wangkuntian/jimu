package outbox

import (
	"context"
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/queue"
	"jimu/internal/contract"
	"jimu/internal/kernel/scheduler"
)

// Wire 装配 outbox 能力：构造事件存储与发布器，把 *Outbox 暴露为端口供 user/auth 写入
// 事件，并注册 outbox_process 定时任务。
//
// publisher=event_bus（默认）：订阅事件总线 outbox:* 主题桥接到裸业务主题，不构造队列；
// publisher=mq：按 queue.type 构造队列客户端（base 行为：仅在此时构造，kafka/rabbitmq
// 会连 broker、缺 broker/topic 即启动失败），消费它注册 MQ 桥接 worker 并把 WorkerPool
// 纳入生命周期。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	// 跨能力校验（原 config.validateCommon 的 outbox.publisher=mq 依赖 queue.type）
	queueCfg := assembly.MustSection[*queue.Config](ctx, queue.ConfigKey)
	if cfg.UsesMQ() && (queueCfg == nil || !queue.SupportsOutboxMQ(queueCfg.Type)) {
		return nil, fmt.Errorf("invalid queue.type %q for outbox.publisher %q", queueTypeOf(queueCfg), cfg.Publisher)
	}

	outboxStore := NewMySQLStore(ctx.DB())
	var publisher Publisher
	switch cfg.Publisher {
	case PublisherMQ:
		qc := *queueCfg
		qc.Redis = ctx.Redis()
		q, err := queue.New(qc)
		if err != nil {
			return nil, fmt.Errorf("init outbox queue: %w", err)
		}
		consumer, ok := q.(queue.Consumer)
		if !ok {
			return nil, fmt.Errorf("queue %s does not implement consumer", queueTypeOf(queueCfg))
		}
		publisher = NewMQPublisher(q)
		pool := queue.NewWorkerPool(queue.DefaultWorkerConfig, consumer, queue.NewMySQLStoreForDB(ctx.DB()))
		RegisterMQWorkers(ctx.EventBus())
		ctx.RegisterComponent(queue.NewWorkerPoolComponent(pool))
	default:
		publisher = NewEventBusPublisher(ctx.EventBus())
		RegisterEventBusBridge(ctx.EventBus(), ctx.Logger())
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
	return nil, nil
}

// queueTypeOf 读取 queue 配置的类型（queue 未启用时为零值）。
func queueTypeOf(cfg *queue.Config) queue.Type {
	if cfg == nil {
		return ""
	}
	return cfg.Type
}
