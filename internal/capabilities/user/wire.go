package user

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/outbox"
	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/contract"
)

// Wire 装配用户能力：以软依赖端口（access 的角色分配、tenant 的配额、outbox 的事件
// 落库）构造用户模块，并把 contract.UserinfoSource 暴露为端口供 mfa/passkey/grpc 消费。
//
// access/tenant 端口名用字面量：tenant 能力 import 本包（迁移依赖），反向 import 会成环。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	roles, _ := ctx.Port("access").(application.UserRoleAssigner)
	quota, _ := ctx.Port("tenant").(application.TenantQuota)
	ob, _ := ctx.Port(outbox.PortName).(*outbox.Outbox)
	mod := New(ctx.DB(), *ctx.Config(), ctx.Redis(), ob).WithRoles(roles).WithQuota(quota)
	if err := ctx.Provide(UserinfoPortName, NewUserinfoSource(infrastructure.NewMysqlRepository(ctx.DB()))); err != nil {
		return nil, fmt.Errorf("provide user info port: %w", err)
	}
	return mod, nil
}
