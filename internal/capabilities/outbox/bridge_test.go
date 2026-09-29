package outbox

import (
	"context"
	"encoding/json"
	"testing"

	"jimu/internal/capabilities/queue"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/logger"

	"github.com/stretchr/testify/assert"
)

func newTestLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
}

func TestBridgeWorkerPublishesStrongTypeToBareTopic(t *testing.T) {
	bus := event.New()
	bus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, uint64(7), evt.UserID)
			assert.Equal(t, "bob", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 7, Username: "bob"})
	evtPayload, _ := json.Marshal(EventPayload{
		ID:          1,
		AggregateID: "user:7",
		EventType:   contract.EventUserCreated,
		Payload:     payload,
	})

	err := BridgeWorker(bus)(context.Background(), string(evtPayload))
	assert.NoError(t, err)
}

func TestBridgeWorkerUnknownTypeErrors(t *testing.T) {
	bus := event.New()
	evtPayload, _ := json.Marshal(EventPayload{
		ID:        1,
		EventType: "order.created",
		Payload:   json.RawMessage(`{}`),
	})
	err := BridgeWorker(bus)(context.Background(), string(evtPayload))
	assert.Error(t, err)
}

func TestBridgeWorkerConversionFailureErrors(t *testing.T) {
	bus := event.New()
	// 已知事件类型但 Payload 不是该强类型的 JSON（如数组）：转换应报错，不发布零值事件
	evtPayload, _ := json.Marshal(EventPayload{
		ID:        1,
		EventType: contract.EventUserCreated,
		Payload:   json.RawMessage(`[1,2,3]`),
	})
	err := BridgeWorker(bus)(context.Background(), string(evtPayload))
	assert.Error(t, err)
}

func TestEventBusBridgePublishesToBareTopic(t *testing.T) {
	bus := event.New()
	bus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, "carol", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})
	RegisterEventBusBridge(bus, newTestLogger())

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 9, Username: "carol"})
	bus.Publish("outbox:"+contract.EventUserCreated, EventPayload{
		ID:        1,
		EventType: contract.EventUserCreated,
		Payload:   payload,
	})
}

func TestRegisterMQWorkersRegistersAll(t *testing.T) {
	// queue 包全局 worker map 无导出清理；本测试只断言三个事件类型可注册后 GetWorker 命中，重复运行幂等
	RegisterMQWorkers(event.New())
	for _, et := range []string{"outbox:user.created", "outbox:user.updated", "outbox:user.deleted"} {
		fn, ok := queue.GetWorker(et)
		assert.True(t, ok, "worker %s not registered", et)
		assert.NotNil(t, fn)
	}
}
