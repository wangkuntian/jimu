package passkey

import (
	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// Wire 装配 WebAuthn 能力：WebAuthn 参数取自 passkey 段，JWT/签发参数取自 auth 段，
// 登录收尾经 auth 端口复用（contract.LoginFinalizer，缺失即降级），返回 passkey 模块。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	finalizer, _ := ctx.Port("auth").(contract.LoginFinalizer)
	users, _ := ctx.Port(contract.UserinfoPortName).(contract.UserinfoSource)
	return New(Deps{
		DB:         ctx.DB(),
		Redis:      ctx.Redis(),
		AuthCfg:    ctx.AuthConfig(),
		Config:     *cfg,
		Users:      users,
		Finalizer:  finalizer,
		FailClosed: ctx.Config().HTTP.Mode == config.HTTPModeRelease,
	}), nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
