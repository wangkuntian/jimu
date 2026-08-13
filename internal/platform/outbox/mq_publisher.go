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

// Publish 发布事件到消息队列。
// 将 Add 时注入事件 Metadata 的 trace 上下文透传到 JobData，供消费者恢复链路。
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
		traceparent, tracestate := traceFromMetadata(e.Metadata)
		job := &queue.JobData{
			ID:          e.ID,
			Type:        "outbox:" + e.EventType,
			Payload:     string(data),
			Traceparent: traceparent,
			Tracestate:  tracestate,
		}
		if err := p.queue.Submit(ctx, job); err != nil {
			return fmt.Errorf("submit outbox event %d: %w", e.ID, err)
		}
	}
	return nil
}

// traceFromMetadata 从事件 Metadata 提取 W3C 追踪字段。
func traceFromMetadata(metadata json.RawMessage) (traceparent, tracestate string) {
	if len(metadata) == 0 {
		return "", ""
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(metadata, &meta); err != nil {
		return "", ""
	}
	tp, _ := meta["traceparent"].(string)
	ts, _ := meta["tracestate"].(string)
	return tp, ts
}

var _ Publisher = (*MQPublisher)(nil)
