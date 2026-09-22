package apikey

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配 API Key 能力：构造管理端模块（/api/v1/admin/apikeys*）。
// 租户归属经 tenant 端口（软依赖，缺失即不校验租户配额）；受保护中间件由本能力提供，
// 但只在启用集没有 auth 时（machine 形态）——auth 已在时 apikey 让位，单提供者规则不变。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// auth 排在 apikey 之前（catalog 顺序），端口存在即表示启用集含 auth。
	return New(ctx.DB(), ctx.Port("tenant"), WithProtectedMiddleware(ctx.Port("auth") == nil)), nil
}
