package outbox

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
)

// TestOutboxMQBoundaryPreservesPayload verifies the outbox side of MQ transport.
func TestOutboxMQBoundaryPreservesPayload(t *testing.T) {
	publisher := &captureMQPublisher{}
	store := &e2eStore{events: []Event{{
		ID:          1,
		AggregateID: "user-1",
		EventType:   "e2e.UserCreated",
		Payload:     json.RawMessage(`{"name":"tom"}`),
	}}}

	count, err := New(store, NewMQPublisher(publisher)).Process(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, []uint64{1}, store.marked)
	require.Len(t, publisher.messages, 1)
	require.Equal(t, "e2e.UserCreated", publisher.messages[0].EventType)

	var event EventPayload
	require.NoError(t, json.Unmarshal([]byte(publisher.messages[0].Payload), &event))
	require.Equal(t, uint64(1), event.ID)
	require.Equal(t, "user-1", event.AggregateID)
	require.JSONEq(t, `{"name":"tom"}`, string(event.Payload))
}

type captureMQPublisher struct{ messages []contract.OutboxMQMessage }

func (p *captureMQPublisher) Publish(_ context.Context, messages ...contract.OutboxMQMessage) error {
	p.messages = append(p.messages, messages...)
	return nil
}

type e2eStore struct {
	events []Event
	marked []uint64
}

func (s *e2eStore) Add(context.Context, interface{}, ...Event) error { return nil }
func (s *e2eStore) FetchUnpublish(context.Context, int) ([]Event, error) {
	return s.events, nil
}
func (s *e2eStore) MarkPublished(_ context.Context, ids []uint64) error {
	s.marked = append(s.marked, ids...)
	return nil
}
func (s *e2eStore) MarkFailed(context.Context, uint64, error) error { return nil }
