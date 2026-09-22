package dataops

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配数据导入导出能力：返回用户导入管理端模块（/api/v1/admin/users/import*）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	return New(ctx.DB()), nil
}
