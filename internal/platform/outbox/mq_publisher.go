package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"jimu/internal/platform/queue"
)

// MQPublisher 发布 Outbox 事件到消息队列，支持跨服务分发
type MQPublisher struct {
	queue queue.Queue
}

// NewMQPublisher 创建 MQ 发布器
func NewMQPublisher(q queue.Queue) *MQPublisher {
	return &MQPublisher{queue: q}
}

// Publish 发布事件到消息队列
func (p *MQPublisher) Publish(ctx context.Context, events ...Event) error {
	for _, e := range events {
		payload := EventPayload{
			ID:          e.ID,
			AggregateID: e.AggregateID,
			EventType:   e.EventType,
			Payload:     e.Payload,
			Metadata:    e.Metadata,
			CreatedAt:   e.CreatedAt,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal outbox event %d: %w", e.ID, err)
		}
		job := &queue.JobData{
			ID:      e.ID,
			Type:    "outbox:" + e.EventType,
			Payload: string(data),
		}
		if err := p.queue.Submit(ctx, job); err != nil {
			return fmt.Errorf("submit outbox event %d: %w", e.ID, err)
		}
	}
	return nil
}

var _ Publisher = (*MQPublisher)(nil)
