package notification

import (
	"context"
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配通知能力：解码三段通知配置，构造 Dispatcher 与 WebSocket Hub，注册各渠道
// （未配置真实发送渠道时用日志型兜底），把 Dispatcher 暴露为端口供 user/auth 消费，
// 把 UserCreatedEmailNotification 领域事件桥接到通知分发，并把 Hub 作为组件纳入生命周期。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	notifCfg, err := Load(ctx.Sections())
	if err != nil {
		return nil, fmt.Errorf("init notification config: %w", err)
	}
	log := ctx.Logger()
	notifier := NewDispatcher()
	// WebSocket Hub（通知渠道 + 实时通信共用）
	wsHub := NewHub()

	// 未配置真实发送渠道时，注册日志型兜底渠道，保证通知链路不报错且可观察
	var emailChannel Notification = NewLogChannel(ChannelEmail, log)
	if notifCfg.Email.Enabled {
		emailChannel = NewEmail(EmailConfig{
			Host:     notifCfg.Email.Host,
			Port:     notifCfg.Email.Port,
			Username: notifCfg.Email.Username,
			Password: notifCfg.Email.Password,
			From:     notifCfg.Email.From,
		})
	}
	notifier.Register(ChannelEmail, emailChannel)

	// 短信：未配置真实发送时注册日志型兜底渠道，保证通知链路不报错且可观察
	var smsChannel Notification = NewLogChannel(ChannelSMS, log)
	if notifCfg.SMS.Enabled {
		smsChannel = NewSMS(SMSConfig{
			Provider:  notifCfg.SMS.Provider,
			APIKey:    notifCfg.SMS.APIKey,
			APISecret: notifCfg.SMS.APISecret,
			SignName:  notifCfg.SMS.SignName,
		})
	}
	notifier.Register(ChannelSMS, smsChannel)

	notifier.Register(ChannelWebSocket, NewWebSocket(wsHub))
	notifier.Register(ChannelWebhook, NewWebhook(WebhookConfig{
		Headers:    map[string]string{},
		SignSecret: notifCfg.Notification.Webhook.SignSecret,
	}, ctx.HTTPClient()))

	if err := ctx.Provide(PortName, notifier); err != nil {
		return nil, fmt.Errorf("provide notification port: %w", err)
	}
	ctx.RegisterComponent(NewHubComponent(wsHub))

	// 注册全局事件处理器：将领域事件桥接到通知系统
	ctx.EventBus().Subscribe(contract.UserCreatedEmailNotification, func(payload interface{}) {
		if msg, ok := payload.(Message); ok {
			if err := notifier.Dispatch(context.Background(), msg); err != nil {
				log.Errorw("notification dispatch failed", "error", err.Error())
			}
		}
	})
	return nil, nil
}
