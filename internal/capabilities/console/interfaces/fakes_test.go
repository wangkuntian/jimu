package interfaces

import (
	"sync"
)

// fakeEventBus 记录发布事件的事件总线 mock
type fakeEventBus struct {
	mu     sync.Mutex
	events []string
}

func (f *fakeEventBus) Subscribe(event string, handler func(payload interface{})) {}
func (f *fakeEventBus) Publish(event string, payload interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}
