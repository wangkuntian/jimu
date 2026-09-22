package passkey

import (
	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/user"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// Wire 装配 WebAuthn 能力：JWT/签发参数取自 auth 段，用户信息经 user.info 端口读取，
// 登录收尾经 auth 端口复用（contract.LoginFinalizer，缺失即降级），返回 passkey 模块。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	finalizer, _ := ctx.Port(authmodule.PortName).(contract.LoginFinalizer)
	users, _ := ctx.Port(user.UserinfoPortName).(contract.UserinfoSource)
	return New(Deps{
		DB:         ctx.DB(),
		Redis:      ctx.Redis(),
		AuthCfg:    *authConfig(ctx),
		Users:      users,
		Finalizer:  finalizer,
		FailClosed: ctx.Config().HTTP.Mode == config.HTTPModeRelease,
	}), nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}
