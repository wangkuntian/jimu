package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
)

func TestOutboxMQFactoryRejectsUnsupportedTypeBeforeConstructingDriver(t *testing.T) {
	called := false
	factory := outboxMQFactory{
		cfg:      Config{Type: "nats"},
		newQueue: func(Config) (Queue, error) { called = true; return nil, nil },
	}
	_, err := factory.StartOutbox(context.Background(), nil, nil)
	require.ErrorContains(t, err, `invalid queue.type "nats"`)
	require.False(t, called)
}

func TestOutboxMQFactoryPropagatesDriverFailure(t *testing.T) {
	factory := outboxMQFactory{
		cfg:      Config{Type: TypeKafka},
		newQueue: func(Config) (Queue, error) { return nil, errors.New("broker unavailable") },
	}
	_, err := factory.StartOutbox(context.Background(), nil, nil)
	require.ErrorContains(t, err, "broker unavailable")
}

func TestOutboxMQFactoryMapsMessageAndRegistersWorker(t *testing.T) {
	q := &outboxMQTestQueue{}
	registered := 0
	var received contract.OutboxMQMessage
	factory := outboxMQFactory{
		cfg:               Config{Type: TypeRedis},
		newQueue:          func(Config) (Queue, error) { return q, nil },
		registerComponent: func(contract.Component) { registered++ },
	}
	publisher, err := factory.StartOutbox(context.Background(), []string{"user.created"}, func(_ context.Context, m contract.OutboxMQMessage) error {
		received = m
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, registered)
	message := contract.OutboxMQMessage{ID: 7, EventType: "user.created", Payload: `{"id":7}`, Traceparent: "trace", Tracestate: "state"}
	require.NoError(t, publisher.Publish(context.Background(), message))
	require.Len(t, q.jobs, 1)
	require.Equal(t, uint64(7), q.jobs[0].ID)
	require.Equal(t, "outbox:user.created", q.jobs[0].Type)
	require.Equal(t, `{"id":7}`, q.jobs[0].Payload)
	require.Equal(t, "trace", q.jobs[0].Traceparent)
	require.Equal(t, "state", q.jobs[0].Tracestate)
	worker, ok := GetWorker("outbox:user.created")
	require.True(t, ok)
	require.NoError(t, worker(context.Background(), q.jobs[0].Payload))
	require.Equal(t, message.EventType, received.EventType)
	require.Equal(t, message.Payload, received.Payload)
}

type outboxMQTestQueue struct{ jobs []*JobData }

func (q *outboxMQTestQueue) Submit(_ context.Context, job *JobData) error {
	q.jobs = append(q.jobs, job)
	return nil
}
func (*outboxMQTestQueue) SubmitDelayed(context.Context, *JobData, time.Duration) error { return nil }
func (*outboxMQTestQueue) MoveDueJobs(context.Context) (int, error)                     { return 0, nil }
func (*outboxMQTestQueue) Consume(context.Context, time.Duration) (*JobData, error)     { return nil, nil }
func (*outboxMQTestQueue) Ack(context.Context, *JobData) error                          { return nil }
func (*outboxMQTestQueue) Nack(context.Context, *JobData) error                         { return nil }
