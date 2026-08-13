package outbox

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// fakeStore 内存 Outbox 存储，捕获 Add 传入的事件
type fakeStore struct {
	added []Event
}

func (f *fakeStore) Add(ctx context.Context, tx interface{}, events ...Event) error {
	f.added = append(f.added, events...)
	return nil
}

func (f *fakeStore) FetchUnpublish(ctx context.Context, limit int) ([]Event, error) { return nil, nil }
func (f *fakeStore) MarkPublished(ctx context.Context, ids []uint64) error          { return nil }
func (f *fakeStore) MarkFailed(ctx context.Context, id uint64, err error) error     { return nil }

// TestOutboxAddInjectsTraceMetadata 验证 Add 将调用方 trace 上下文注入事件 Metadata
func TestOutboxAddInjectsTraceMetadata(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	fs := &fakeStore{}
	o := New(fs, nil)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	err := o.Add(ctx, nil, Event{ID: 1, EventType: "UserCreated", Payload: json.RawMessage(`{}`)})
	assert.NoError(t, err)
	require.Len(t, fs.added, 1)
	assert.Contains(t, string(fs.added[0].Metadata), "traceparent")
}

// TestOutboxAddKeepsExistingMetadata 验证注入不覆盖调用方已有 Metadata
func TestOutboxAddKeepsExistingMetadata(t *testing.T) {
	fs := &fakeStore{}
	o := New(fs, nil)

	err := o.Add(context.Background(), nil, Event{
		ID:        1,
		EventType: "UserCreated",
		Metadata:  json.RawMessage(`{"source":"import"}`),
	})
	assert.NoError(t, err)
	require.Len(t, fs.added, 1)
	assert.JSONEq(t, `{"source":"import"}`, string(fs.added[0].Metadata))
}

// TestMQPublisherPropagatesTrace 验证 MQ 发布器将事件 Metadata 的 trace 透传到 JobData
func TestMQPublisherPropagatesTrace(t *testing.T) {
	fq := &fakeQueue{}
	p := NewMQPublisher(fq)

	events := []Event{{
		ID:        1,
		EventType: "UserCreated",
		Payload:   json.RawMessage(`{}`),
		Metadata:  json.RawMessage(`{"traceparent":"00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01"}`),
	}}

	err := p.Publish(context.Background(), events...)
	require.NoError(t, err)
	require.Len(t, fq.submitted, 1)
	assert.Equal(t, "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01", fq.submitted[0].Traceparent)
}
