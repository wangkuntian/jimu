package mfa

import (
	"fmt"

	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/user"
	"jimu/internal/contract"
)

// Wire 装配 TOTP 二次验证 + 可信设备能力：JWT/签发参数取自 auth 段，用户信息经
// user.info 端口读取（软依赖，缺失时仅影响 otpauth account 兜底），把 MFAVerifier
// 暴露为端口供 auth 消费。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := authConfig(ctx)
	users, _ := ctx.Port(user.UserinfoPortName).(contract.UserinfoSource)
	mod := New(ctx.DB(), Config{
		JWTSecret:         cfg.JWTSecret,
		JWTPreviousSecret: cfg.JWTPreviousSecret,
		Issuer:            cfg.Issuer,
		AccessExpireMin:   cfg.AccessExpireMin,
		RefreshExpireDay:  cfg.RefreshExpireDay,
		TrustedDeviceDays: cfg.TrustedDeviceDays,
	}, users)
	if err := ctx.Provide(PortName, mod.Service()); err != nil {
		return nil, fmt.Errorf("provide mfa port: %w", err)
	}
	return mod, nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}
