package notification

import (
	"context"
	"encoding/json"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/dara"
)

// SMSConfig 短信配置
type SMSConfig struct {
	Provider  string `mapstructure:"provider"` // aliyun, tencent
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	SignName  string `mapstructure:"sign_name"` // 短信签名
}

// SMS 短信通知实现
type SMS struct {
	config   SMSConfig
	endpoint string // 测试用端点覆盖；空则使用阿里云默认地址
}

// NewSMS 创建短信通知
func NewSMS(config SMSConfig) *SMS {
	return &SMS{config: config}
}

// setEndpoint 覆盖短信服务端点（仅测试用）
func (s *SMS) setEndpoint(ep string) {
	s.endpoint = ep
}

func (s *SMS) Send(ctx context.Context, msg Message) error {
	switch s.config.Provider {
	case "aliyun":
		return s.sendAliyun(ctx, msg)
	case "tencent":
		return s.sendTencent(ctx, msg)
	default:
		return fmt.Errorf("SMS provider %q not configured", s.config.Provider)
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
	cfg := &openapi.Config{
		AccessKeyId:     dara.String(s.config.APIKey),
		AccessKeySecret: dara.String(s.config.APISecret),
		Endpoint:        dara.String("dysmsapi.aliyuncs.com"),
	}
	if s.endpoint != "" {
		cfg.Protocol = dara.String("http")
		cfg.Endpoint = dara.String(s.endpoint)
	}
	client, err := dysmsapi.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("init aliyun sms client: %w", err)
	}

	param, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("marshal sms template param: %w", err)
	}
	req := &dysmsapi.SendSmsRequest{}
	req.SetPhoneNumbers(msg.To).
		SetSignName(s.config.SignName).
		SetTemplateCode(msg.TemplateID).
		SetTemplateParam(string(param))

	resp, err := client.SendSms(req)
	if err != nil {
		return fmt.Errorf("aliyun sms send: %w", err)
	}
	if resp.Body == nil {
		return fmt.Errorf("aliyun sms send: empty response")
	}
	if code := dara.StringValue(resp.Body.Code); code != "OK" {
		return fmt.Errorf("aliyun sms send failed: code=%s message=%s", code, dara.StringValue(resp.Body.Message))
	}
	return nil
}

func (s *SMS) sendTencent(ctx context.Context, msg Message) error {
	return fmt.Errorf("tencent SMS provider not configured")
}

var _ Notification = (*SMS)(nil)
