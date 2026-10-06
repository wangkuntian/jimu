package mfa

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配 TOTP 二次验证 + 可信设备能力：JWT 参数取自 auth 段，设备策略取自 mfa 段，用户信息经
// user.info 端口读取（软依赖，缺失时仅影响 otpauth account 兜底），把 MFAVerifier
// 暴露为端口供 auth 消费。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	authCfg := ctx.AuthConfig()
	settings := assembly.MustSection[*Settings](ctx, ConfigKey)
	if settings == nil {
		settings = &Settings{}
	}
	if job, ok := newMFARetentionJob(ctx.DB(), settings.Retention, ctx.Logger()); ok {
		if err := ctx.RegisterJob(job); err != nil {
			return nil, err
		}
	}
	users, _ := ctx.Port(contract.UserinfoPortName).(contract.UserinfoSource)
	mod := New(ctx.DB(), Config{
		JWTSecret:         authCfg.JWTSecret,
		JWTPreviousSecret: authCfg.JWTPreviousSecret,
		Issuer:            authCfg.Issuer,
		AccessExpireMin:   authCfg.AccessExpireMin,
		RefreshExpireDay:  authCfg.RefreshExpireDay,
		TrustedDeviceDays: settings.TrustedDeviceDays,
	}, users)
	if err := ctx.Provide(PortName, mod.Service()); err != nil {
		return nil, fmt.Errorf("provide mfa port: %w", err)
	}
	return mod, nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
