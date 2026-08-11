package notification

import (
	"context"

	"jimu/internal/platform/logger"
)

// LogChannel 日志型通知渠道
// 未配置真实发送渠道时兜底，把通知写入日志，避免链路报 "channel not registered"。
// SMTP/SMS 等真实实现就绪后可移除注册，改走对应渠道。
type LogChannel struct {
	channel Channel
	log     *logger.Logger
}

// NewLogChannel 创建日志型通知渠道
func NewLogChannel(ch Channel, log *logger.Logger) *LogChannel {
	return &LogChannel{channel: ch, log: log}
}

func (l *LogChannel) Send(ctx context.Context, msg Message) error {
	l.log.WithContext(ctx).Info("notification delivered to log channel",
		"channel", msg.Channel,
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.Body,
		"data", msg.Data,
	)
	return nil
}

func (l *LogChannel) SendBatch(ctx context.Context, msgs []Message) error {
	for _, msg := range msgs {
		if err := l.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (l *LogChannel) Channel() Channel {
	return l.channel
}

var _ Notification = (*LogChannel)(nil)
