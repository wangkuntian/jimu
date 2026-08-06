package notification

import (
	"context"
	"fmt"
)

// SMSConfig 短信配置
type SMSConfig struct {
	Provider string `mapstructure:"provider"` // aliyun, tencent, twilio
	APIKey   string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	SignName string `mapstructure:"sign_name"` // 短信签名
}

// SMS 短信通知实现
type SMS struct {
	config SMSConfig
}

// NewSMS 创建短信通知
func NewSMS(config SMSConfig) *SMS {
	return &SMS{config: config}
}

func (s *SMS) Send(ctx context.Context, msg Message) error {
	switch s.config.Provider {
	case "aliyun":
		return s.sendAliyun(ctx, msg)
	case "tencent":
		return s.sendTencent(ctx, msg)
	default:
		return fmt.Errorf("SMS provider %q not implemented", s.config.Provider)
	}
}

func (s *SMS) SendBatch(ctx context.Context, msgs []Message) error {
	for _, msg := range msgs {
		if err := s.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SMS) Channel() Channel {
	return ChannelSMS
}

func (s *SMS) sendAliyun(ctx context.Context, msg Message) error {
	// TODO: 引入 aliyun-dysmsapi-sdk
	return fmt.Errorf("Aliyun SMS not implemented yet")
}

func (s *SMS) sendTencent(ctx context.Context, msg Message) error {
	// TODO: 引入 tencentcloud-sdk-go
	return fmt.Errorf("Tencent SMS not implemented yet")
}

var _ Notification = (*SMS)(nil)
