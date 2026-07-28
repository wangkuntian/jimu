package event

import (
	"sync"
	"testing"
	"time"
)

func TestEventBus_SubscribeAndPublish(t *testing.T) {
	bus := New()
	var received string

	bus.Subscribe("test", func(payload interface{}) {
		received = payload.(string)
	})

	bus.Publish("test", "hello")

	if received != "hello" {
		t.Errorf("expected 'hello', got '%s'", received)
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := New()
	var count int
	var mu sync.Mutex

	bus.Subscribe("multi", func(payload interface{}) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	bus.Subscribe("multi", func(payload interface{}) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.Publish("multi", nil)

	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2 handlers called, got %d", count)
	}
}

func TestEventBus_PublishAsync(t *testing.T) {
	bus := New()
	var received string
	done := make(chan struct{})

	bus.Subscribe("async", func(payload interface{}) {
		received = payload.(string)
		close(done)
	})

	bus.PublishAsync("async", "world")

	select {
	case <-done:
		if received != "world" {
			t.Errorf("expected 'world', got '%s'", received)
		}
	case <-time.After(time.Second):
		t.Error("async handler not called within 1 second")
	}
}

func TestEventBus_Clear(t *testing.T) {
	bus := New()
	var called bool

	bus.Subscribe("clear", func(payload interface{}) {
		called = true
	})
	bus.Clear("clear")
	bus.Publish("clear", nil)

	if called {
		t.Error("handler should not be called after Clear")
	}
}
