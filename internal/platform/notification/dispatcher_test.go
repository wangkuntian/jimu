package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNotification struct {
	sendErr    error
	batchErr   error
	sendCalls  int
	batchCalls int
	channel    Channel
}

func (m *mockNotification) Send(_ context.Context, _ Message) error {
	m.sendCalls++
	return m.sendErr
}

func (m *mockNotification) SendBatch(_ context.Context, _ []Message) error {
	m.batchCalls++
	return m.batchErr
}

func (m *mockNotification) Channel() Channel { return m.channel }

func TestDispatcherDispatch(t *testing.T) {
	d := NewDispatcher().(*dispatcher)
	reg := &mockNotification{channel: ChannelEmail}
	d.Register(ChannelEmail, reg)

	// 成功
	require.NoError(t, d.Dispatch(context.Background(), Message{Channel: ChannelEmail}))
	assert.Equal(t, 1, reg.sendCalls)

	// 未注册渠道
	err := d.Dispatch(context.Background(), Message{Channel: ChannelSMS})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")

	// 渠道内部错误
	failing := &mockNotification{sendErr: errors.New("boom"), channel: ChannelWebhook}
	d.Register(ChannelWebhook, failing)
	require.Error(t, d.Dispatch(context.Background(), Message{Channel: ChannelWebhook}))
}

func TestDispatcherDispatchBatch(t *testing.T) {
	d := NewDispatcher().(*dispatcher)
	email := &mockNotification{channel: ChannelEmail}
	d.Register(ChannelEmail, email)

	msgs := []Message{
		{Channel: ChannelEmail, To: "a@x.com"},
		{Channel: ChannelEmail, To: "b@x.com"},
		{Channel: ChannelSMS, To: "13800000000"}, // 未注册渠道 → 错误
	}
	err := d.DispatchBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, 1, email.batchCalls)

	// 全注册 + 无错误
	d.Register(ChannelSMS, &mockNotification{channel: ChannelSMS})
	require.NoError(t, d.DispatchBatch(context.Background(), msgs))
}
