package contract

import (
	"encoding/json"
	"net/http"

	"jimu/internal/kernel/auth"
)

// AdminWebSocketMessage 是管理端 WebSocket 消息的纯数据视图。
// Payload 由端口实现编码，能力消费者不依赖 ws 的消息类型。
type AdminWebSocketMessage struct {
	Type    string
	Channel string
	Payload json.RawMessage
}

// AdminWebSocketPresence 是管理端在线状态的纯数据视图。
type AdminWebSocketPresence struct {
	Status string
}

// AdminWebSocket 是 console 所需的最小 WebSocket 端口。
// 实现负责连接 Hub、频道和 presence 的生命周期；console 只注册路由和转发管理请求。
type AdminWebSocket interface {
	Handler(jwt *auth.JWT) http.HandlerFunc
	SendToUser(userID uint64, message AdminWebSocketMessage) error
	BroadcastToChannel(channel string, message AdminWebSocketMessage) error
	Broadcast(message AdminWebSocketMessage) error
	Presence(userID uint64) (AdminWebSocketPresence, bool)
	OnlineUsers() []uint64
	OnlineCount() int
	Connections(userID uint64) int
}
