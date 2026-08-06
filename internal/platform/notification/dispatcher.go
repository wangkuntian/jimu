package notification

import (
	"context"
	"fmt"
	"sync"
)

// dispatcher 通知调度器实现
type dispatcher struct {
	mu        sync.RWMutex
	channels  map[Channel]Notification
}

// NewDispatcher 创建通知调度器
func NewDispatcher() Dispatcher {
	return &dispatcher{
		channels: make(map[Channel]Notification),
	}
}

func (d *dispatcher) Register(ch Channel, n Notification) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.channels[ch] = n
}

func (d *dispatcher) Dispatch(ctx context.Context, msg Message) error {
	d.mu.RLock()
	n, ok := d.channels[msg.Channel]
	d.mu.RUnlock()

	if !ok {
		return fmt.Errorf("notification channel %q not registered", msg.Channel)
	}

	return n.Send(ctx, msg)
}

func (d *dispatcher) DispatchBatch(ctx context.Context, msgs []Message) error {
	// 按渠道分组
	grouped := make(map[Channel][]Message)
	for _, msg := range msgs {
		grouped[msg.Channel] = append(grouped[msg.Channel], msg)
	}

	// 并发发送各渠道
	var wg sync.WaitGroup
	errCh := make(chan error, len(grouped))

	for ch, group := range grouped {
		wg.Add(1)
		go func(ch Channel, msgs []Message) {
			defer wg.Done()

			d.mu.RLock()
			n, ok := d.channels[ch]
			d.mu.RUnlock()

			if !ok {
				errCh <- fmt.Errorf("notification channel %q not registered", ch)
				return
			}

			if err := n.SendBatch(ctx, msgs); err != nil {
				errCh <- fmt.Errorf("channel %q: %w", ch, err)
			}
		}(ch, group)
	}

	wg.Wait()
	close(errCh)

	// 收集错误
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

var _ Dispatcher = (*dispatcher)(nil)
