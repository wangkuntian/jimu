package tenant

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配租户能力：从 auth 段的 provisioning 配置构造开通式注册输入视图，把
// TenantQuota 与 TenantProvisioner 暴露为端口（user/auth/apikey 经它们消费）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	mod := New(ctx.DB(), provisioningConfig(ctx.AuthConfig()))
	if err := ctx.Provide(PortName, mod.Quota()); err != nil {
		return nil, fmt.Errorf("provide tenant port: %w", err)
	}
	if err := ctx.Provide(ProvisionerPortName, contract.TenantProvisionerFactory(mod)); err != nil {
		return nil, fmt.Errorf("provide tenant provisioner port: %w", err)
	}
	return mod, nil
}

// provisioningConfig 把 auth 段的 provisioning 配置映射为 tenant 自有的输入视图。
// tenant 被 auth 依赖、不得 import auth 的类型混用，故两边类型独立，在此显式转换（P2.1 裁定）。
func provisioningConfig(cfg contract.AuthConfig) ProvisioningConfig {
	p := cfg.Provisioning
	roles := make([]ProvisionRoleTemplate, 0, len(p.Roles))
	for _, role := range p.Roles {
		perms := make([]ProvisionPermission, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			perms = append(perms, ProvisionPermission{Resource: perm.Resource, Action: perm.Action})
		}
		roles = append(roles, ProvisionRoleTemplate{
			Name:        role.Name,
			Description: role.Description,
			Permissions: perms,
		})
	}
	return ProvisioningConfig{Enabled: p.Enabled, OwnerRole: p.OwnerRole, Roles: roles}
}
