package console

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
)

// Wire 装配管理控制台能力：提供平台级视图与 /api/v1/admin/* 准入中间件，
// JWT 由 auth 段的签发参数构造（auth 未启用时取零值，控制台仅暴露平台视图）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	authCfg := ctx.AuthConfig()
	wsPort, _ := ctx.Port("ws").(contract.AdminWebSocket)
	return New(cfg.Version, cfg.Environment, ctx.Redis(), ctx.DB(),
		auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay),
		ctx.EventBus(), middleware.IPAllowlist(cfg.Security.AdminIPAllowlist), wsPort), nil
}
