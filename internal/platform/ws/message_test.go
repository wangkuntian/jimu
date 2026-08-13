package ws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	payload := NotificationPayload{Title: "hi", Content: "hello", Level: "info"}
	msg, err := NewMessage(TypeNotification, BuildUserChannel(1), payload)
	require.NoError(t, err)
	assert.Equal(t, TypeNotification, msg.Type)
	assert.Equal(t, "user:1", msg.Channel)
	assert.False(t, msg.Time.IsZero())

	var got NotificationPayload
	require.NoError(t, json.Unmarshal(msg.Payload, &got))
	assert.Equal(t, payload, got)
}

func TestNewMessageMarshalError(t *testing.T) {
	_, err := NewMessage(TypeNotification, "broadcast", make(chan int))
	assert.Error(t, err)
}

func TestWSMessageEncodeRoundTrip(t *testing.T) {
	msg := &WSMessage{
		Type:    TypeChat,
		Channel: "room:abc",
		Payload: json.RawMessage(`{"content":"hi"}`),
	}
	data, err := msg.Encode()
	require.NoError(t, err)

	var got WSMessage
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, TypeChat, got.Type)
	assert.Equal(t, "room:abc", got.Channel)
}

func TestWSMessageDecodePayload(t *testing.T) {
	msg := &WSMessage{Payload: json.RawMessage(`{"channels":["room:a","room:b"]}`)}
	var sub SubscribePayload
	require.NoError(t, msg.DecodePayload(&sub))
	assert.Equal(t, []string{"room:a", "room:b"}, sub.Channels)
}

func TestWSMessageDecodePayloadError(t *testing.T) {
	msg := &WSMessage{Payload: json.RawMessage(`{invalid`)}
	var sub SubscribePayload
	assert.Error(t, msg.DecodePayload(&sub))
}

func TestBuildUserChannel(t *testing.T) {
	assert.Equal(t, "user:123", BuildUserChannel(123))
}

func TestBuildRoomChannel(t *testing.T) {
	assert.Equal(t, "room:abc", BuildRoomChannel("abc"))
}
