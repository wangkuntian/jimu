package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"jimu/internal/platform/httpclient"
)

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	// 默认 Headers
	Headers map[string]string `mapstructure:"headers"`
	// SignSecret HMAC-SHA256 载荷签名密钥；非空时发送 X-Jimu-Timestamp + X-Jimu-Signature，
	// 供回调消费者验真与防篡改。为空则不签名。
	SignSecret string `mapstructure:"sign_secret"`
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
	if w.config.SignSecret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set("X-Jimu-Timestamp", ts)
		req.Header.Set("X-Jimu-Signature", signPayload(w.config.SignSecret, ts, body))
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

// signPayload 计算 HMAC-SHA256 签名：hex(hmac(secret, timestamp + "." + body))。
// 时间戳纳入签名防止重放，消费者按 X-Jimu-Timestamp 限时校验。
func signPayload(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

var _ Notification = (*Webhook)(nil)
