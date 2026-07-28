package event

import (
	"sync"
)

// Handler 事件处理函数
type Handler func(payload interface{})

// EventBus 内存事件总线
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// New 创建事件总线
func New() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe 订阅事件
func (b *EventBus) Subscribe(event string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[event] = append(b.handlers[event], handler)
}

// Publish 发布事件（同步执行）
func (b *EventBus) Publish(event string, payload interface{}) {
	b.mu.RLock()
	handlers := b.handlers[event]
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(payload)
	}
}

// PublishAsync 发布事件（异步执行）
func (b *EventBus) PublishAsync(event string, payload interface{}) {
	b.mu.RLock()
	handlers := b.handlers[event]
	b.mu.RUnlock()

	for _, handler := range handlers {
		go handler(payload)
	}
}

// Clear 清除指定事件的所有订阅
func (b *EventBus) Clear(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, event)
}
