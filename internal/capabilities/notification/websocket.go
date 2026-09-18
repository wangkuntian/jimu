package notification

import (
	"context"
	"encoding/json"
	"sync"
)

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	// 心跳间隔（秒）
	HeartbeatInterval int `mapstructure:"heartbeat_interval"`
	// 写超时（秒）
	WriteTimeout int `mapstructure:"write_timeout"`
	// 最大消息大小（字节）
	MaxMessageSize int64 `mapstructure:"max_message_size"`
}

// Connection WebSocket 连接接口
type Connection interface {
	Send(data []byte) error
	Close() error
	UserID() string
}

// Hub WebSocket 连接管理器
type Hub struct {
	mu          sync.RWMutex
	connections map[string]Connection // userID -> Connection
	broadcast   chan Message
	register    chan Registration
	unregister  chan string
}

// Registration 注册信息
type Registration struct {
	UserID     string
	Connection Connection
}

// NewHub 创建 WebSocket Hub
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]Connection),
		broadcast:   make(chan Message, 256),
		register:    make(chan Registration, 128),
		unregister:  make(chan string, 128),
	}
}

// Run 启动 Hub（消费广播消息）
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reg := <-h.register:
			h.mu.Lock()
			h.connections[reg.UserID] = reg.Connection
			h.mu.Unlock()
		case userID := <-h.unregister:
			h.mu.Lock()
			if conn, ok := h.connections[userID]; ok {
				_ = conn.Close()
				delete(h.connections, userID)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			conn, ok := h.connections[msg.To]
			h.mu.RUnlock()
			if ok {
				data, _ := json.Marshal(msg)
				_ = conn.Send(data)
			}
		}
	}
}

// Register 注册连接
func (h *Hub) Register(userID string, conn Connection) {
	h.register <- Registration{UserID: userID, Connection: conn}
}

// Unregister 注销连接
func (h *Hub) Unregister(userID string) {
	h.unregister <- userID
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID string, msg Message) {
	msg.To = userID
	msg.Channel = ChannelWebSocket
	select {
	case h.broadcast <- msg:
	default:
		// 广播通道满，丢弃消息（可改为日志记录）
	}
}

// SendToUsers 发送消息给多个用户
func (h *Hub) SendToUsers(userIDs []string, msg Message) {
	for _, userID := range userIDs {
		h.SendToUser(userID, msg)
	}
}

// OnlineCount 返回在线连接数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// IsOnline 检查用户是否在线
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.connections[userID]
	return ok
}

// WebSocket WebSocket 通知实现
type WebSocket struct {
	hub *Hub
}

// NewWebSocket 创建 WebSocket 通知
func NewWebSocket(hub *Hub) *WebSocket {
	return &WebSocket{hub: hub}
}

func (w *WebSocket) Send(ctx context.Context, msg Message) error {
	w.hub.SendToUser(msg.To, msg)
	return nil
}

func (w *WebSocket) SendBatch(ctx context.Context, msgs []Message) error {
	for _, msg := range msgs {
		if err := w.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (w *WebSocket) Channel() Channel {
	return ChannelWebSocket
}

var _ Notification = (*WebSocket)(nil)
