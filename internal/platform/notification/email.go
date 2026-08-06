package notification

import (
	"context"
	"fmt"
)

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	// 可选：使用第三方 API（SendGrid、SES 等）
	Provider string `mapstructure:"provider"` // smtp, sendgrid, ses
	APIKey   string `mapstructure:"api_key"`
}

// Email 邮件通知实现
type Email struct {
	config EmailConfig
}

// NewEmail 创建邮件通知
func NewEmail(config EmailConfig) *Email {
	return &Email{config: config}
}

func (e *Email) Send(ctx context.Context, msg Message) error {
	if e.config.Provider == "smtp" || e.config.Provider == "" {
		return e.sendSMTP(ctx, msg)
	}
	return fmt.Errorf("email provider %q not implemented", e.config.Provider)
}

func (e *Email) SendBatch(ctx context.Context, msgs []Message) error {
	for _, msg := range msgs {
		if err := e.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (e *Email) Channel() Channel {
	return ChannelEmail
}

func (e *Email) sendSMTP(ctx context.Context, msg Message) error {
	// TODO: 引入 gomail 或其他 SMTP 库实现
	// 这里提供接口占位
	return fmt.Errorf("SMTP email not implemented yet (use gomail.v2)")
}

var _ Notification = (*Email)(nil)
