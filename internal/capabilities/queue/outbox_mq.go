package queue

import (
	"context"
	"fmt"
	"sync"

	"jimu/internal/contract"

	"gorm.io/gorm"
)

type outboxMQFactory struct {
	cfg               Config
	db                *gorm.DB
	newQueue          func(Config) (Queue, error)
	registerComponent func(contract.Component)
	mu                sync.Mutex
	started           bool
}

func (f *outboxMQFactory) StartOutbox(ctx context.Context, eventTypes []string, handler contract.OutboxMQHandler) (contract.OutboxMQPublisher, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		return nil, fmt.Errorf("outbox MQ factory already started")
	}
	if !SupportsOutboxMQ(f.cfg.Type) {
		return nil, fmt.Errorf("invalid queue.type %q", f.cfg.Type)
	}
	newQueue := f.newQueue
	if newQueue == nil {
		newQueue = New
	}
	cfg := f.cfg
	q, err := newQueue(cfg)
	if err != nil {
		return nil, err
	}
	consumer, ok := q.(Consumer)
	if !ok {
		return nil, fmt.Errorf("queue %s does not implement consumer", f.cfg.Type)
	}
	if handler == nil {
		return nil, fmt.Errorf("outbox MQ handler is nil")
	}
	for _, eventType := range eventTypes {
		eventType := eventType
		RegisterWorker("outbox:"+eventType, func(ctx context.Context, payload string) error {
			return handler(ctx, contract.OutboxMQMessage{EventType: eventType, Payload: payload})
		})
	}
	pool := NewWorkerPool(DefaultWorkerConfig, consumer, NewMySQLStoreForDB(f.db))
	if f.registerComponent != nil {
		f.registerComponent(NewWorkerPoolComponent(pool))
	}
	f.started = true
	return &outboxMQPublisher{queue: q}, nil
}

type outboxMQPublisher struct{ queue Queue }

func (p *outboxMQPublisher) Publish(ctx context.Context, messages ...contract.OutboxMQMessage) error {
	for _, message := range messages {
		if err := p.queue.Submit(ctx, &JobData{
			ID:          message.ID,
			Type:        "outbox:" + message.EventType,
			Payload:     message.Payload,
			Traceparent: message.Traceparent,
			Tracestate:  message.Tracestate,
		}); err != nil {
			return fmt.Errorf("submit outbox event %d: %w", message.ID, err)
		}
	}
	return nil
}

var _ contract.OutboxMQFactory = (*outboxMQFactory)(nil)
var _ contract.OutboxMQPublisher = (*outboxMQPublisher)(nil)
