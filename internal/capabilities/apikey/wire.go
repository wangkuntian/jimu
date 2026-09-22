package apikey

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配 API Key 能力：构造管理端模块（/api/v1/admin/apikeys*）。
// 租户归属经 tenant 端口（软依赖，缺失即不校验租户配额）；认证中间件由需要保护
// 的业务路由按需挂载。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	return New(ctx.DB(), ctx.Port("tenant")), nil
}
