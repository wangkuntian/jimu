package feature

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配 Feature Flag 能力：返回管理端模块（/api/v1/admin/features*）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	return New(ctx.DB()), nil
}
