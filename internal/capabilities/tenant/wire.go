package tenant

import (
	"fmt"

	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/contract"
)

// Wire 装配租户能力：从 auth 段的 provisioning 配置构造开通式注册输入视图，把
// TenantQuota 与 TenantProvisioner 暴露为端口（user/auth/apikey 经它们消费）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	mod := New(ctx.DB(), provisioningConfig(authConfig(ctx)))
	if err := ctx.Provide(PortName, mod.Quota()); err != nil {
		return nil, fmt.Errorf("provide tenant port: %w", err)
	}
	if err := ctx.Provide(ProvisionerPortName, mod.Provisioner()); err != nil {
		return nil, fmt.Errorf("provide tenant provisioner port: %w", err)
	}
	return mod, nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}

// provisioningConfig 把 auth 段的 provisioning 配置映射为 tenant 自有的输入视图。
// tenant 被 auth 依赖、不得 import auth 的类型混用，故两边类型独立，在此显式转换（P2.1 裁定）。
func provisioningConfig(cfg *authmodule.Config) ProvisioningConfig {
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
