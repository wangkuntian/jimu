package event

import (
	"log"
	"sync"
)

// Handler 事件处理函数
type Handler func(payload interface{})

// maxAsyncHandlers 异步事件并发处理上限，防止事件风暴打满 goroutine
const maxAsyncHandlers = 1024

// EventBus 内存事件总线（实现 contract.EventBus 接口）
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	sem      chan struct{} // 异步事件并发信号量
}

// New 创建事件总线
func New() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
		sem:      make(chan struct{}, maxAsyncHandlers),
	}
}

// Subscribe 订阅事件（实现 contract.EventBus 接口）
func (b *EventBus) Subscribe(event string, handler func(payload interface{})) {
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

// PublishAsync 发布事件（异步执行）。
// 并发受信号量限制，handler panic 会被恢复并记录日志，不影响进程。
func (b *EventBus) PublishAsync(event string, payload interface{}) {
	b.mu.RLock()
	handlers := b.handlers[event]
	b.mu.RUnlock()

	for _, handler := range handlers {
		b.sem <- struct{}{} // 并发超限时阻塞发布方，背压保护
		go func(h Handler) {
			defer func() { <-b.sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("event: handler panic on %q recovered: %v", event, r)
				}
			}()
			h(payload)
		}(handler)
	}
}

// Clear 清除指定事件的所有订阅
func (b *EventBus) Clear(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, event)
}
