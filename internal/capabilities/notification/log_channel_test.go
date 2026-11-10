package notification

import (
	"context"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogChannelSendNoError(t *testing.T) {
	l := logger.New(config.LogConfig{Level: "error"})
	ch := NewLogChannel(ChannelEmail, l)

	err := ch.Send(context.Background(), Message{
		Channel: ChannelEmail,
		To:      "user@example.com",
		Subject: "Welcome",
		Body:    "Hi",
	})
	require.NoError(t, err)
	assert.Equal(t, ChannelEmail, ch.Channel())
}

func TestLogChannelSendBatchNoError(t *testing.T) {
	l := logger.New(config.LogConfig{Level: "error"})
	ch := NewLogChannel(ChannelEmail, l)

	err := ch.SendBatch(context.Background(), []Message{
		{Channel: ChannelEmail, To: "a@example.com"},
		{Channel: ChannelEmail, To: "b@example.com"},
	})
	require.NoError(t, err)
}
