package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
)

// adminWebSocket 持有管理端实时连接的唯一 hub、频道和在线状态。
type adminWebSocket struct {
	hub      *ClientHub
	presence *PresenceManager
	channels *ChannelManager
	cancel   context.CancelFunc
}

func newAdminWebSocket() *adminWebSocket {
	presence := NewPresenceManager()
	channels := NewChannelManager()
	return &adminWebSocket{
		hub:      NewClientHub(presence, channels),
		presence: presence,
		channels: channels,
	}
}

func (w *adminWebSocket) Handler(jwt *auth.JWT) http.HandlerFunc {
	return WSHandler(w.hub, jwt, w.presence, w.channels)
}

func (w *adminWebSocket) SendToUser(userID uint64, message contract.AdminWebSocketMessage) error {
	msg, err := adminMessage(message)
	if err != nil {
		return err
	}
	w.hub.SendToUser(userID, msg)
	return nil
}

func (w *adminWebSocket) BroadcastToChannel(channel string, message contract.AdminWebSocketMessage) error {
	msg, err := adminMessage(message)
	if err != nil {
		return err
	}
	w.hub.BroadcastToChannel(channel, msg)
	return nil
}

func (w *adminWebSocket) Broadcast(message contract.AdminWebSocketMessage) error {
	msg, err := adminMessage(message)
	if err != nil {
		return err
	}
	w.hub.Broadcast(msg)
	return nil
}

func adminMessage(message contract.AdminWebSocketMessage) (*WSMessage, error) {
	payload := message.Payload
	if payload == nil {
		payload = json.RawMessage("null")
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("invalid message payload")
	}
	return &WSMessage{
		Type: message.Type, Channel: message.Channel,
		Payload: payload, Time: time.Now(),
	}, nil
}

func (w *adminWebSocket) Presence(userID uint64) (contract.AdminWebSocketPresence, bool) {
	p, ok := w.presence.GetPresence(userID)
	if !ok {
		return contract.AdminWebSocketPresence{}, false
	}
	return contract.AdminWebSocketPresence{Status: p.Status}, true
}

func (w *adminWebSocket) OnlineUsers() []uint64 { return w.presence.OnlineUsers() }
func (w *adminWebSocket) OnlineCount() int      { return w.presence.OnlineCount() }
func (w *adminWebSocket) Connections(userID uint64) int {
	return w.hub.GetUserConnections(userID)
}

func (w *adminWebSocket) Start(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.hub.Run(ctx)
	return nil
}

func (w *adminWebSocket) Stop(context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

var _ contract.AdminWebSocket = (*adminWebSocket)(nil)
var _ contract.Component = (*adminWebSocket)(nil)
