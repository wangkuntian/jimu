package apidocs

import (
	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// Wire 装配 API 文档能力：release 模式下不注册任何路由；返回模块把 Swagger UI
// 挂到根路由的 /swagger/* （公开挂载，不套用受保护中间件）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	return NewModule(ctx.Config().HTTP.Mode != config.HTTPModeRelease), nil
}
