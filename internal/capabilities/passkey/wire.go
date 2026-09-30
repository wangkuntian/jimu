package passkey

import (
	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// Wire 装配 WebAuthn 能力：JWT/签发参数取自 auth 段，用户信息经 user.info 端口读取，
// 登录收尾经 auth 端口复用（contract.LoginFinalizer，缺失即降级），返回 passkey 模块。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	finalizer, _ := ctx.Port("auth").(contract.LoginFinalizer)
	users, _ := ctx.Port(contract.UserinfoPortName).(contract.UserinfoSource)
	return New(Deps{
		DB:         ctx.DB(),
		Redis:      ctx.Redis(),
		AuthCfg:    ctx.AuthConfig(),
		Users:      users,
		Finalizer:  finalizer,
		FailClosed: ctx.Config().HTTP.Mode == config.HTTPModeRelease,
	}), nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
