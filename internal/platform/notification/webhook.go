package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"jimu/internal/platform/httpclient"
)

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	// 默认 Headers
	Headers map[string]string `mapstructure:"headers"`
}

// Webhook Webhook 通知实现
type Webhook struct {
	config WebhookConfig
	client *httpclient.Client
}

// NewWebhook 创建 Webhook 通知（复用统一出站 client：超时/重试/trace 注入）
func NewWebhook(config WebhookConfig, client *httpclient.Client) *Webhook {
	return &Webhook{config: config, client: client}
}

func (w *Webhook) Send(ctx context.Context, msg Message) error {
	if w.client == nil {
		return fmt.Errorf("webhook http client not configured")
	}

	payload := map[string]interface{}{
		"to":      msg.To,
		"subject": msg.Subject,
		"body":    msg.Body,
		"data":    msg.Data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msg.To, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.config.Headers {
		req.Header.Set(k, v)
	}

	// 统一 client 负责网络错误与 5xx 重试；4xx 不重试，此处转业务错误
	resp, err := w.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (w *Webhook) SendBatch(ctx context.Context, msgs []Message) error {
	for _, msg := range msgs {
		if err := w.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (w *Webhook) Channel() Channel {
	return ChannelWebhook
}

var _ Notification = (*Webhook)(nil)
