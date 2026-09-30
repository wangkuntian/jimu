package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
	"jimu/internal/kernel/event"
)

type fakeMQFactory struct {
	calls   int
	types   []string
	handler contract.OutboxMQHandler
	pub     contract.OutboxMQPublisher
	err     error
}

func (f *fakeMQFactory) StartOutbox(_ context.Context, types []string, handler contract.OutboxMQHandler) (contract.OutboxMQPublisher, error) {
	f.calls++
	f.types = types
	f.handler = handler
	return f.pub, f.err
}

func TestSelectPublisherEventBusDoesNotStartMQ(t *testing.T) {
	factory := &fakeMQFactory{}
	publisher, err := selectPublisher(context.Background(), Config{Publisher: PublisherEventBus}, factory, event.New(), newTestLogger())
	require.NoError(t, err)
	require.IsType(t, &EventBusPublisher{}, publisher)
	require.Zero(t, factory.calls)
}

func TestSelectPublisherMQStartsOnceWithBridge(t *testing.T) {
	mq := &fakeMQPublisher{}
	factory := &fakeMQFactory{pub: mq}
	bus := event.New()
	publisher, err := selectPublisher(context.Background(), Config{Publisher: PublisherMQ}, factory, bus, newTestLogger())
	require.NoError(t, err)
	require.IsType(t, &MQPublisher{}, publisher)
	require.Equal(t, 1, factory.calls)
	require.Contains(t, factory.types, contract.EventUserCreated)
	require.NotNil(t, factory.handler)
}

func TestSelectPublisherMQFailsClosed(t *testing.T) {
	_, err := selectPublisher(context.Background(), Config{Publisher: PublisherMQ}, nil, event.New(), newTestLogger())
	require.ErrorContains(t, err, "outbox MQ port")

	factory := &fakeMQFactory{err: errors.New("broker unavailable")}
	_, err = selectPublisher(context.Background(), Config{Publisher: PublisherMQ}, factory, event.New(), newTestLogger())
	require.ErrorContains(t, err, "broker unavailable")
	require.Equal(t, 1, factory.calls)
}

type fakeMQPublisher struct{ messages []contract.OutboxMQMessage }

func (p *fakeMQPublisher) Publish(_ context.Context, messages ...contract.OutboxMQMessage) error {
	p.messages = append(p.messages, messages...)
	return nil
}
