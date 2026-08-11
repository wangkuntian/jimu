package app

import (
	"context"
	"encoding/json"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/event"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/outbox"

	"github.com/stretchr/testify/assert"
)

// fakeContainer 最小 Container：EventBus + Logger（registerEventBusBridge 需 Logger）
func fakeContainer() *Container {
	return &Container{
		EventBus: event.New(),
		Logger:   newTestLogger(),
	}
}

func newTestLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
}

func TestBridgeWorkerPublishesStrongTypeToBareTopic(t *testing.T) {
	c := fakeContainer()
	c.EventBus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, uint64(7), evt.UserID)
			assert.Equal(t, "bob", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 7, Username: "bob"})
	evtPayload, _ := json.Marshal(outbox.EventPayload{
		ID:          1,
		AggregateID: "user:7",
		EventType:   contract.EventUserCreated,
		Payload:     payload,
	})

	err := bridgeFn(c)(context.Background(), string(evtPayload))
	assert.NoError(t, err)
}

func TestBridgeWorkerUnknownTypeErrors(t *testing.T) {
	c := fakeContainer()
	evtPayload, _ := json.Marshal(outbox.EventPayload{
		ID:        1,
		EventType: "order.created",
		Payload:   json.RawMessage(`{}`),
	})
	err := bridgeFn(c)(context.Background(), string(evtPayload))
	assert.Error(t, err)
}

func TestEventBusBridgePublishesToBareTopic(t *testing.T) {
	c := fakeContainer()
	c.EventBus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, "carol", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})
	registerEventBusBridge(c)

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 9, Username: "carol"})
	c.EventBus.Publish("outbox:"+contract.EventUserCreated, outbox.EventPayload{
		ID:        1,
		EventType: contract.EventUserCreated,
		Payload:   payload,
	})
}
