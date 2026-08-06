package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	// 默认 Headers
	Headers map[string]string `mapstructure:"headers"`
}

// Webhook Webhook 通知实现
type Webhook struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhook 创建 Webhook 通知
func NewWebhook(config WebhookConfig) *Webhook {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &Webhook{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

func (w *Webhook) Send(ctx context.Context, msg Message) error {
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

	var lastErr error
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, msg.To, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		for k, v := range w.config.Headers {
			req.Header.Set(k, v)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook failed after %d retries: %w", w.config.MaxRetries+1, lastErr)
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
