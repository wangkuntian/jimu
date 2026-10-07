package tenant

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配租户能力：从 auth 段的 provisioning 配置构造开通式注册输入视图，把
// TenantQuota 与 TenantProvisioner 暴露为端口（user/auth/apikey 经它们消费）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	mod := New(ctx.DB(), cfg.Provisioning)
	if err := ctx.Provide(PortName, mod.Quota()); err != nil {
		return nil, fmt.Errorf("provide tenant port: %w", err)
	}
	if err := ctx.Provide(ProvisionerPortName, contract.TenantProvisionerFactory(mod)); err != nil {
		return nil, fmt.Errorf("provide tenant provisioner port: %w", err)
	}
	return mod, nil
}
