package access

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配访问控制能力：构造角色/权限/用户角色服务，把 UserRoleAssigner 暴露为端口，
// 供排在其后的 user 消费（缺此端口时 user 的角色分配降级为未配置）。
//
// tenant 端口名用字面量：tenant 能力 import 本包（迁移依赖），本包反向 import 会成环。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	mod := New(ctx.DB(), ctx.Port("tenant"))
	if err := ctx.Provide(PortName, mod.UserRoleAssigner()); err != nil {
		return nil, fmt.Errorf("provide access port: %w", err)
	}
	return mod, nil
}
