package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"jimu/internal/capabilities/queue"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/logger"
)

// PortName outbox 能力对外提供的端口名：*Outbox（user/auth 写入事件经它落库）。
const PortName = "outbox"

// eventTypeConverters 按事件类型将 outbox 内层 Payload 还原为强类型事件。
// 返回 error：载荷与事件类型不匹配时拒绝发布，避免零值事件被静默发出。
var eventTypeConverters = map[string]func(json.RawMessage) (interface{}, error){
	contract.EventUserCreated: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserCreatedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserUpdated: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserUpdatedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserDeleted: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserDeletedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserLoggedIn: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserLoggedInEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
}

// BridgeWorker 返回把 outbox 载荷反序列化并发布强类型事件到裸业务主题的 worker。
func BridgeWorker(bus *event.EventBus) queue.WorkerFunc {
	return func(_ context.Context, payload string) error {
		var evt EventPayload
		if err := json.Unmarshal([]byte(payload), &evt); err != nil {
			return fmt.Errorf("unmarshal outbox event: %w", err)
		}
		conv, ok := eventTypeConverters[evt.EventType]
		if !ok {
			return fmt.Errorf("no converter for outbox event type: %s", evt.EventType)
		}
		strong, err := conv(evt.Payload)
		if err != nil {
			return fmt.Errorf("convert outbox event %s: %w", evt.EventType, err)
		}
		bus.Publish(evt.EventType, strong)
		return nil
	}
}

// RegisterMQWorkers 注册 MQ 消费端的 outbox 桥接 worker。
func RegisterMQWorkers(bus *event.EventBus) {
	for eventType := range eventTypeConverters {
		queue.RegisterWorker("outbox:"+eventType, BridgeWorker(bus))
	}
}

// RegisterEventBusBridge 订阅全局总线 outbox:* 主题，转强类型后发布到裸业务主题
// （publisher=event_bus 模式）。
func RegisterEventBusBridge(bus *event.EventBus, log *logger.Logger) {
	for eventType := range eventTypeConverters {
		eventType := eventType
		bus.Subscribe("outbox:"+eventType, func(payload interface{}) {
			evt, ok := payload.(EventPayload)
			if !ok {
				log.Error("outbox bridge: unexpected payload type")
				return
			}
			conv, ok := eventTypeConverters[evt.EventType]
			if !ok {
				log.Errorw("outbox bridge: unknown event type", "type", evt.EventType)
				return
			}
			strong, err := conv(evt.Payload)
			if err != nil {
				log.Errorw("outbox bridge: convert event failed", "type", evt.EventType, "error", err.Error())
				return
			}
			bus.Publish(evt.EventType, strong)
		})
	}
}
