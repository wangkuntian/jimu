package captcha

import (
	"fmt"
	"time"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// PortName 验证码能力对外提供的端口名：contract.CaptchaVerifier。
const PortName = "captcha"

// Wire 装配验证码能力：构造服务与模块，并把 contract.CaptchaVerifier 暴露为端口，
// 供 auth 在软依赖可用时消费（缺失时端口取回 nil，auth 跳过验证码校验）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// captcha 启用时其配置段必已按启用集解码（Descriptor.Configs 声明，设计 §8）。
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	mod := New(ctx.Redis(), time.Duration(cfg.TTLMin)*time.Minute, cfg.Enabled)
	if err := ctx.Provide(PortName, mod.Service()); err != nil {
		return nil, fmt.Errorf("provide captcha port: %w", err)
	}
	return mod, nil
}
