package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"jimu/internal/platform/auth"
	"jimu/internal/shared/errors"

	"github.com/gorilla/websocket"
)

// 配置常量
const (
	// 写超时
	writeWait = 10 * time.Second
	// 心跳间隔
	pongWait = 60 * time.Second
	// 发送 ping 周期（必须小于 pongWait）
	pingPeriod = (pongWait * 9) / 10
	// 最大消息大小（64KB）
	maxMessageSize = 65536
	// 发送通道缓冲
	sendBufferSize = 256
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域（生产环境应限制来源）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 表示一个 WebSocket 连接
type Client struct {
	hub    *ClientHub
	conn   *websocket.Conn
	send   chan []byte
	userID uint64
	// 订阅的频道列表
	mu       sync.RWMutex
	channels map[string]bool
	// 连接 ID
	connID string
	// 最后活跃时间
	lastActive time.Time
}

// ReadPump 处理来自客户端的消息
func (c *Client) ReadPump(ctx context.Context, jwtUtil *auth.JWT, presence *PresenceManager, channels *ChannelManager, onRegister func(*Client)) {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		presence.Heartbeat(c.userID)
		c.mu.Lock()
		c.lastActive = time.Now()
		c.mu.Unlock()
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			// 忽略正常关闭错误，仅处理异常关闭
			if !websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
			) {
				break
			}
			// 异常关闭，直接退出
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid message format")
			continue
		}

		c.handleMessage(ctx, jwtUtil, presence, channels, &msg)
	}
}

// handleMessage 处理单条消息
func (c *Client) handleMessage(ctx context.Context, jwtUtil *auth.JWT, presence *PresenceManager, channels *ChannelManager, msg *WSMessage) {
	switch msg.Type {
	case TypePing:
		ping := &WSMessage{
			Type:    TypePong,
			Channel: msg.Channel,
			Time:    time.Now(),
		}
		if data, err := json.Marshal(PingPayload{Timestamp: time.Now().UnixMilli()}); err == nil {
			ping.Payload = data
		}
		c.sendMessage(ping)

	case TypeSubscribe:
		var sub SubscribePayload
		if err := msg.DecodePayload(&sub); err != nil {
			c.sendError("invalid subscribe payload")
			return
		}
		c.mu.Lock()
		for _, ch := range sub.Channels {
			c.channels[ch] = true
			channels.Subscribe(c.connID, ch)
		}
		c.mu.Unlock()

	case TypeUnsubscribe:
		var sub SubscribePayload
		if err := msg.DecodePayload(&sub); err != nil {
			c.sendError("invalid unsubscribe payload")
			return
		}
		c.mu.Lock()
		for _, ch := range sub.Channels {
			delete(c.channels, ch)
			channels.Unsubscribe(c.connID, ch)
		}
		c.mu.Unlock()

	case TypePresence:
		var p PresencePayload
		if err := msg.DecodePayload(&p); err == nil && p.Status == StatusTyping {
			presence.SetTyping(c.userID)
			// 广播 typing 状态到用户频道
			c.hub.broadcastPresence(c.userID, StatusTyping)
		}

	case TypeChat:
		var chat ChatPayload
		if err := msg.DecodePayload(&chat); err != nil {
			c.sendError("invalid chat payload")
			return
		}
		// 将聊天消息广播到目标频道
		c.hub.broadcastToChannel(msg.Channel, msg)

	default:
		c.sendError("unknown message type: " + msg.Type)
	}
}

// WritePump 向客户端发送消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub 关闭通道
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// 批量发送排队中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendMessage 发送消息到客户端
func (c *Client) sendMessage(msg *WSMessage) {
	data, err := msg.Encode()
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// 发送缓冲区满，关闭连接
		c.hub.unregister <- c
	}
}

// sendError 发送错误消息
func (c *Client) sendError(errMsg string) {
	msg := &WSMessage{
		Type:    "error",
		Channel: "",
		Time:    time.Now(),
	}
	if data, err := json.Marshal(map[string]string{"error": errMsg}); err == nil {
		msg.Payload = data
	}
	c.sendMessage(msg)
}

// Channels 返回当前订阅的频道列表
func (c *Client) Channels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		result = append(result, ch)
	}
	return result
}

// ClientHub 管理所有 WebSocket 连接
type ClientHub struct {
	// 所有连接
	mu      sync.RWMutex
	clients map[string]*Client // connID -> client

	// 注册/注销通道
	register   chan *Client
	unregister chan *Client

	// 按用户索引（支持多端登录）
	userIndex map[uint64]map[string]bool // userID -> set of connIDs

	// 频道广播
	broadcast chan *broadcastMsg

	// 依赖
	presence *PresenceManager
	channels *ChannelManager
}

type broadcastMsg struct {
	channel string
	message *WSMessage
}

// NewClientHub 创建 ClientHub
func NewClientHub(presence *PresenceManager, channels *ChannelManager) *ClientHub {
	return &ClientHub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 128),
		unregister: make(chan *Client, 128),
		userIndex:  make(map[uint64]map[string]bool),
		broadcast:  make(chan *broadcastMsg, 256),
		presence:   presence,
		channels:   channels,
	}
}

// Run 启动 hub 主循环
func (h *ClientHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case bm := <-h.broadcast:
			h.handleBroadcast(bm)
		}
	}
}

func (h *ClientHub) handleRegister(client *Client) {
	h.mu.Lock()
	h.clients[client.connID] = client
	if h.userIndex[client.userID] == nil {
		h.userIndex[client.userID] = make(map[string]bool)
	}
	h.userIndex[client.userID][client.connID] = true
	h.mu.Unlock()

	// 更新 presence
	p := h.presence.Online(client.userID, client.connID)
	_ = p

	// 订阅用户个人频道
	userChannel := BuildUserChannel(client.userID)
	client.mu.Lock()
	client.channels[userChannel] = true
	client.mu.Unlock()
	h.channels.Subscribe(client.connID, userChannel)

	// 订阅广播频道
	client.mu.Lock()
	client.channels[ChannelBroadcast] = true
	client.mu.Unlock()
	h.channels.Subscribe(client.connID, ChannelBroadcast)

	// 广播用户上线
	h.broadcastPresence(client.userID, StatusOnline)
}

func (h *ClientHub) handleUnregister(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client.connID]; ok {
		delete(h.clients, client.connID)
		close(client.send)
		if conns, ok := h.userIndex[client.userID]; ok {
			delete(conns, client.connID)
			if len(conns) == 0 {
				delete(h.userIndex, client.userID)
			}
		}
	}
	h.mu.Unlock()

	// 取消所有频道订阅
	channels := client.Channels()
	h.channels.UnsubscribeAll(client.connID, channels)

	// 用户下线（仅当无其他连接时）
	h.mu.RLock()
	hasOtherConnections := len(h.userIndex[client.userID]) > 0
	h.mu.RUnlock()
	if !hasOtherConnections {
		h.presence.Offline(client.userID)
		h.broadcastPresence(client.userID, StatusOffline)
	}

	_ = client.conn.Close()
}

func (h *ClientHub) handleBroadcast(bm *broadcastMsg) {
	subscribers := h.channels.GetSubscribers(bm.channel)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, connID := range subscribers {
		if client, ok := h.clients[connID]; ok {
			select {
			case client.send <- mustEncode(bm.message):
			default:
				// 发送缓冲区满，准备关闭
				go func(c *Client) {
					h.unregister <- c
				}(client)
			}
		}
	}
}

// broadcastPresence 广播用户在线状态变化
func (h *ClientHub) broadcastPresence(userID uint64, status string) {
	payload := PresencePayload{
		UserID:    userID,
		Status:    status,
		Timestamp: time.Now().UnixMilli(),
	}
	payloadData, _ := json.Marshal(payload)
	msg := &WSMessage{
		Type:    TypePresence,
		Channel: BuildUserChannel(userID),
		Payload: payloadData,
		Time:    time.Now(),
	}
	// 广播到该用户频道和广播频道
	h.broadcastToChannel(BuildUserChannel(userID), msg)
	h.broadcastToChannel(ChannelBroadcast, msg)
}

// broadcastToChannel 广播消息到频道
func (h *ClientHub) broadcastToChannel(channel string, msg *WSMessage) {
	select {
	case h.broadcast <- &broadcastMsg{channel: channel, message: msg}:
	default:
		// 广播通道满，丢弃
	}
}

// SendToUser 发送消息给指定用户
func (h *ClientHub) SendToUser(userID uint64, msg *WSMessage) {
	channel := BuildUserChannel(userID)
	h.broadcastToChannel(channel, msg)
}

// SendToUsers 发送消息给多个用户
func (h *ClientHub) SendToUsers(userIDs []uint64, msg *WSMessage) {
	for _, id := range userIDs {
		h.SendToUser(id, msg)
	}
}

// Broadcast 广播消息到所有连接
func (h *ClientHub) Broadcast(msg *WSMessage) {
	h.broadcastToChannel(ChannelBroadcast, msg)
}

// OnlineCount 返回在线连接数
func (h *ClientHub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// OnlineUsers 返回在线用户 ID 列表
func (h *ClientHub) OnlineUsers() []uint64 {
	return h.presence.OnlineUsers()
}

// GetUserConnections 返回用户的连接数
func (h *ClientHub) GetUserConnections(userID uint64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userIndex[userID])
}

// mustEncode 编码消息（忽略错误）
func mustEncode(msg *WSMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

// WSHandler 创建 WebSocket HTTP 处理函数
func WSHandler(hub *ClientHub, jwtUtil *auth.JWT, presence *PresenceManager, channels *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 从查询参数获取 JWT token
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		// 2. 验证 JWT token
		claims, err := jwtUtil.Parse(token, auth.TokenTypeAccess)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// 3. 升级到 WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "upgrade failed", http.StatusBadRequest)
			return
		}

		// 4. 创建客户端
		client := &Client{
			hub:        hub,
			conn:       conn,
			send:       make(chan []byte, sendBufferSize),
			userID:     claims.UserID,
			connID:     strconv.FormatUint(claims.UserID, 10) + ":" + strconv.FormatInt(time.Now().UnixNano(), 10),
			channels:   make(map[string]bool),
			lastActive: time.Now(),
		}

		// 5. 注册到 hub
		hub.register <- client

		// 6. 启动读写 goroutine
		ctx := r.Context()
		go client.WritePump()
		go client.ReadPump(ctx, jwtUtil, presence, channels, nil)
	}
}

// Ensure errors import is used
var _ = errors.New
