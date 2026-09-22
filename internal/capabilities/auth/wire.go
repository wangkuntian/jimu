package authmodule

import (
	"errors"
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// errProvisioningRequiresPublicRegistration 开通式注册要求公开注册（auth 段跨字段校验）。
var errProvisioningRequiresPublicRegistration = errors.New("auth.provisioning.enabled requires auth.public_registration")

// Wire 装配 auth 能力：校验 auth 段的跨字段约束，从端口取回可选/必需依赖
// （captcha/mfa/tenant/outbox/notification/encryption/breach，缺失即降级），
// 把 LoginFinalizer 暴露为端口供 passkey 复用登录收尾。
//
// 端口名用字面量：tenant/mfa/breach 能力 import 本包（配置视图/校验），反向 import 会成环。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力未启用/未装配时端口取回
	// nil，登录/注册跳过验证码校验。
	captchaVerifier, _ := ctx.Port("captcha").(contract.CaptchaVerifier)
	mod := New(ctx.DB(), ctx.Redis(), *cfg,
		ctx.Config().HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier,
		ctx.Port("outbox"), ctx.Port("notification"), ctx.Port("encryption"),
		ctx.Port("tenant"), ctx.Port("mfa"), ctx.Port("tenant.provisioner"), ctx.Port("breach"))
	if err := ctx.Provide(PortName, mod.Finalizer()); err != nil {
		return nil, fmt.Errorf("provide auth port: %w", err)
	}
	return mod, nil
}

// validateConfig 承担 auth 段的跨字段校验：provisioning 与 public_registration 同段
// （设计 §8 ¶2），开通式注册必须同时开启公开注册。provisioning 的语义归 tenant，
// P2.1 裁定把这条校验留在装配侧（与 outbox.publisher 依赖 queue.type 同理）。
func validateConfig(cfg *Config) error {
	if cfg.Provisioning.Enabled && !cfg.PublicRegistration {
		return errProvisioningRequiresPublicRegistration
	}
	return nil
}
