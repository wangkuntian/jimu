package outbox

import (
	"context"
	"encoding/json"

	"jimu/internal/platform/event"
)

// EventBusPublisher 基于内存事件总线的发布器
type EventBusPublisher struct {
	bus *event.EventBus
}

// NewEventBusPublisher 创建事件总线发布器
func NewEventBusPublisher(bus *event.EventBus) *EventBusPublisher {
	return &EventBusPublisher{bus: bus}
}

// Publish 发布事件到内存事件总线
func (p *EventBusPublisher) Publish(ctx context.Context, events ...Event) error {
	for _, e := range events {
		payload := EventPayload{
			ID:          e.ID,
			AggregateID: e.AggregateID,
			EventType:   e.EventType,
			Payload:     e.Payload,
			Metadata:    e.Metadata,
			CreatedAt:   e.CreatedAt,
		}
		p.bus.Publish("outbox:"+e.EventType, payload)
	}
	return nil
}

// EventPayload 发布到事件总线的载荷
type EventPayload struct {
	ID          uint64          `json:"id"`
	AggregateID string          `json:"aggregate_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   interface{}     `json:"created_at"`
}

var _ Publisher = (*EventBusPublisher)(nil)
