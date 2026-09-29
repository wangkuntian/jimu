package notification

import (
	"context"
)

// Channel 通知渠道
type Channel string

const (
	ChannelEmail     Channel = "email"
	ChannelSMS       Channel = "sms"
	ChannelWebhook   Channel = "webhook"
	ChannelWebSocket Channel = "websocket"
)

// Message 通知消息
type Message struct {
	Channel Channel // 渠道
	To      string  // 收件人（邮箱/手机号/URL/用户ID）
	Subject string  // 主题（邮件/短信标题）
	Body    string  // 内容
	// 扩展字段
	TemplateID string            // 模板 ID
	Data       map[string]string // 模板变量
	Metadata   map[string]string // 自定义元数据
}

// Notification 通知接口
type Notification interface {
	// Send 发送单条消息
	Send(ctx context.Context, msg Message) error

	// SendBatch 批量发送
	SendBatch(ctx context.Context, msgs []Message) error

	// Channel 返回支持的渠道
	Channel() Channel
}

// Dispatcher 通知调度器
type Dispatcher interface {
	// Register 注册通知渠道
	Register(ch Channel, n Notification)

	// Dispatch 根据消息渠道自动分发
	Dispatch(ctx context.Context, msg Message) error

	// DispatchBatch 批量分发
	DispatchBatch(ctx context.Context, msgs []Message) error
}
