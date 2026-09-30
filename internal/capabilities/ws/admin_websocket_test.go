package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/app"
	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminWebSocketPortRoutesAndMessages(t *testing.T) {
	port := newAdminWebSocket()
	message := contract.AdminWebSocketMessage{
		Type: "notification", Channel: "room:1", Payload: json.RawMessage(`{"title":"hi"}`),
	}
	require.NoError(t, port.SendToUser(7, message))
	userMessage := <-port.hub.broadcast
	assert.Equal(t, "user:7", userMessage.channel)
	assert.Equal(t, "room:1", userMessage.message.Channel)
	assert.JSONEq(t, `{"title":"hi"}`, string(userMessage.message.Payload))

	require.NoError(t, port.BroadcastToChannel("room:1", message))
	assert.Equal(t, "room:1", (<-port.hub.broadcast).channel)
	require.NoError(t, port.Broadcast(message))
	assert.Equal(t, ChannelBroadcast, (<-port.hub.broadcast).channel)

	message.Payload = json.RawMessage("bad")
	require.ErrorContains(t, port.Broadcast(message), "invalid message payload")
}

func TestAdminWebSocketPortPresence(t *testing.T) {
	port := newAdminWebSocket()
	_, ok := port.Presence(9)
	assert.False(t, ok)
	port.presence.Online(9, "conn-1")
	p, ok := port.Presence(9)
	require.True(t, ok)
	assert.Equal(t, "online", p.Status)
	assert.Equal(t, 1, port.OnlineCount())
	assert.Equal(t, []uint64{9}, port.OnlineUsers())
}

func TestAdminWebSocketPortHandler(t *testing.T) {
	port := newAdminWebSocket()
	jwt := auth.New("test-secret", "test-issuer", 60, 7)
	h := port.Handler(jwt)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/ws", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing token")

	require.NoError(t, port.Start(context.Background()))
	defer func() { require.NoError(t, port.Stop(context.Background())) }()
	srv := httptest.NewServer(h)
	defer srv.Close()
	token, err := jwt.GenerateAccess(9, 0, "session")
	require.NoError(t, err)
	conn, resp, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(srv.URL, "http")+"?token="+token, nil)
	require.NoError(t, err)
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	require.Eventually(t, func() bool { return port.Connections(9) == 1 }, time.Second, 10*time.Millisecond)
}

func TestWSWireProvidesPortAndLifecycle(t *testing.T) {
	a := assembly.Assembly{Name: "ws-test", Capabilities: []assembly.Capability{{Descriptor: Descriptor, Wire: Wire, Ungated: true}}}
	ctx, modules, err := assembly.WireFor(&app.Container{}, nil, nil, a, []contract.Descriptor{Descriptor})
	require.NoError(t, err)
	assert.Empty(t, modules)
	_, ok := ctx.Port(PortName).(contract.AdminWebSocket)
	assert.True(t, ok)
	require.Len(t, ctx.Components(), 1)
	require.NoError(t, ctx.Components()[0].Start(context.Background()))
	require.NoError(t, ctx.Components()[0].Stop(context.Background()))
}
