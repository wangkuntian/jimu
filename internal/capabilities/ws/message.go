package ws

import (
	"encoding/json"
	"fmt"
	"time"
)

// 消息类型常量
const (
	TypeNotification = "notification" // 系统通知
	TypeChat         = "chat"         // 即时消息
	TypePresence     = "presence"     // 在线状态
	TypePing         = "ping"         // 心跳 ping
	TypePong         = "pong"         // 心跳 pong
	TypeSubscribe    = "subscribe"    // 订阅频道
	TypeUnsubscribe  = "unsubscribe"  // 取消订阅
)

// 频道前缀
const (
	ChannelUserPrefix = "user:"
	ChannelRoomPrefix = "room:"
	ChannelBroadcast  = "broadcast"
)

// WSMessage WebSocket 消息协议
type WSMessage struct {
	Type    string          `json:"type"`    // notification/chat/presence/ping/pong
	Channel string          `json:"channel"` // user:123 / room:abc / broadcast
	Payload json.RawMessage `json:"payload"`
	Time    time.Time       `json:"time"`
}

// PresencePayload 在线状态载荷
type PresencePayload struct {
	UserID    uint64 `json:"user_id"`
	Status    string `json:"status"`    // online/offline/typing
	Timestamp int64  `json:"timestamp"` // 毫秒时间戳
}

// ChatPayload 即时消息载荷
type ChatPayload struct {
	From    uint64 `json:"from"`
	To      string `json:"to"` // user:123 / room:abc
	Content string `json:"content"`
}

// NotificationPayload 通知载荷
type NotificationPayload struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   string `json:"level"` // info/warn/error
}

// PingPayload 心跳请求
type PingPayload struct {
	Timestamp int64 `json:"timestamp"`
}

// PongPayload 心跳响应
type PongPayload struct {
	Timestamp int64 `json:"timestamp"`
}

// SubscribePayload 订阅频道
type SubscribePayload struct {
	Channels []string `json:"channels"`
}

// NewMessage 创建消息
func NewMessage(msgType, channel string, payload interface{}) (*WSMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return &WSMessage{
		Type:    msgType,
		Channel: channel,
		Payload: data,
		Time:    time.Now(),
	}, nil
}

// Encode 编码消息为 JSON 字节
func (m *WSMessage) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodePayload 将 Payload 解码到指定类型
func (m *WSMessage) DecodePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// BuildUserChannel 构建用户频道名
func BuildUserChannel(userID uint64) string {
	return fmt.Sprintf("%s%d", ChannelUserPrefix, userID)
}

// BuildRoomChannel 构建房间频道名
func BuildRoomChannel(roomID string) string {
	return fmt.Sprintf("%s%s", ChannelRoomPrefix, roomID)
}
