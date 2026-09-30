package interfaces

import (
	"net/http"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	"github.com/stretchr/testify/require"
)

type fakeAdminWebSocket struct {
	handler     http.HandlerFunc
	sendUser    uint64
	sendChannel string
	sendMode    string
	message     contract.AdminWebSocketMessage
	presences   map[uint64]contract.AdminWebSocketPresence
	connections map[uint64]int
	users       []uint64
}

func (f *fakeAdminWebSocket) Handler(_ *auth.JWT) http.HandlerFunc { return f.handler }

func (f *fakeAdminWebSocket) SendToUser(userID uint64, message contract.AdminWebSocketMessage) error {
	f.sendMode, f.sendUser, f.message = "user", userID, message
	return nil
}

func (f *fakeAdminWebSocket) BroadcastToChannel(channel string, message contract.AdminWebSocketMessage) error {
	f.sendMode, f.sendChannel, f.message = "channel", channel, message
	return nil
}

func (f *fakeAdminWebSocket) Broadcast(message contract.AdminWebSocketMessage) error {
	f.sendMode, f.message = "broadcast", message
	return nil
}

func (f *fakeAdminWebSocket) Presence(userID uint64) (contract.AdminWebSocketPresence, bool) {
	p, ok := f.presences[userID]
	return p, ok
}

func (f *fakeAdminWebSocket) OnlineUsers() []uint64 { return f.users }
func (f *fakeAdminWebSocket) OnlineCount() int      { return len(f.users) }
func (f *fakeAdminWebSocket) Connections(userID uint64) int {
	return f.connections[userID]
}

func TestAdminWSHandlerAcceptsContractPort(t *testing.T) {
	var port contract.AdminWebSocket = &fakeAdminWebSocket{}
	h := NewAdminWSHandler(port)
	require.NotNil(t, h)
}
