package contract

import (
	"encoding/json"
	"net/http"
)

// AdminWSFactory 构造管理端 WebSocket 的 HTTP handler。
//
// 用不透明构造函数而不是带认证对象参数的接口：contract 包因此不必
// import kernel/auth，否则 0 能力项目也会被拖入 JWT、会话与 Redis。
// ws 能力在 wire 时把自己的 JWT 实例闭包进这个函数。
type AdminWSFactory func() http.HandlerFunc

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
	Handler() AdminWSFactory
	SendToUser(userID uint64, message AdminWebSocketMessage) error
	BroadcastToChannel(channel string, message AdminWebSocketMessage) error
	Broadcast(message AdminWebSocketMessage) error
	Presence(userID uint64) (AdminWebSocketPresence, bool)
	OnlineUsers() []uint64
	OnlineCount() int
	Connections(userID uint64) int
}
