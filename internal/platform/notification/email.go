package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
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

// sendSMTP 通过标准 SMTP 协议发送邮件（net/smtp，纯 stdlib）。
// 支持明文、LOGIN/PLAIN 认证与 STARTTLS。上下文取消会中断连接。
func (e *Email) sendSMTP(ctx context.Context, msg Message) error {
	if e.config.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if msg.To == "" {
		return fmt.Errorf("email recipient is empty")
	}

	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	// net/smtp 不支持 context，用 net.Dialer 建立可取消的连接
	d := &net.Dialer{}

	// 建立连接（支持 context 取消）
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect smtp %s: %w", addr, err)
	}
	c, err := smtp.NewClient(conn, e.config.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// STARTTLS 加密（端口 587 常用）
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: e.config.Host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	// 认证
	if e.config.Username != "" {
		auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	// 发件人/收件人
	from := e.config.From
	if from == "" {
		from = e.config.Username
	}
	if from == "" {
		return fmt.Errorf("smtp from address not configured")
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	// 支持分号分隔的多个收件人
	recipients := strings.Split(msg.To, ";")
	for _, rcpt := range recipients {
		rcpt = strings.TrimSpace(rcpt)
		if rcpt == "" {
			continue
		}
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	// 写邮件内容
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	header := buildEmailHeaders(from, recipients, msg)
	if _, err := fmt.Fprintf(w, "%s\r\n%s\r\n", header, msg.Body); err != nil {
		_ = w.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close email body: %w", err)
	}
	return c.Quit()
}

// buildEmailHeaders 构造邮件 MIME 头
func buildEmailHeaders(from string, recipients []string, msg Message) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	return b.String()
}

var _ Notification = (*Email)(nil)

var _ Notification = (*Email)(nil)
