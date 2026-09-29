package console

import (
	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
)

// Wire 装配管理控制台能力：提供平台级视图与 /api/v1/admin/* 准入中间件，
// JWT 由 auth 段的签发参数构造（auth 未启用时取零值，控制台仅暴露平台视图）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	authCfg := authConfig(ctx)
	return New(cfg.Version, cfg.Environment, ctx.Redis(), ctx.DB(),
		auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay),
		ctx.EventBus(), middleware.IPAllowlist(cfg.Security.AdminIPAllowlist)), nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}
