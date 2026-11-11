package authmodule

import (
	"errors"
	"fmt"
	"time"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/auth/authstore"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// errProvisioningRequiresPublicRegistration tenant 开通式注册要求公开注册。
var errProvisioningRequiresPublicRegistration = errors.New("tenant.provisioning.enabled requires auth.public_registration")

// Wire 装配 auth 能力：校验 auth 段的跨字段约束，从端口取回可选/必需依赖
// （captcha/mfa/tenant/outbox/notification/encryption/breach，缺失即降级），
// 把 LoginFinalizer 暴露为端口供 passkey 复用登录收尾。
//
// 端口名按能力声明读取；可选能力缺席时保留既有降级行为。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	factory, _ := ctx.Port("tenant.provisioner").(contract.TenantProvisionerFactory)
	provisioningEnabled := factory != nil && factory.Enabled()
	if err := validateConfig(cfg, provisioningEnabled); err != nil {
		return nil, err
	}
	users, ok := ctx.Port("user.account").(contract.AccountRepository)
	if !ok {
		return nil, fmt.Errorf("auth requires user.account port")
	}
	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力未启用/未装配时端口取回
	// nil，登录/注册跳过验证码校验。
	captchaVerifier, _ := ctx.Port("captcha").(contract.CaptchaVerifier)
	var provisioner contract.TenantProvisioner
	if provisioningEnabled {
		roles, ok := ctx.Port("access.provisioning").(contract.ProvisioningRoleStore)
		if !ok {
			return nil, fmt.Errorf("auth provisioning requires access.provisioning port")
		}
		provisioner = factory.Build(users, roles)
	}
	driver := cfg.Driver
	if driver == "" {
		driver = authstore.DefaultDriver()
	}
	stores, err := authstore.New(driver, authstore.Options{
		RedisClient:  ctx.Redis(),
		ResetCodeTTL: time.Duration(cfg.ResetCodeTTLMin) * time.Minute,
	})
	if err != nil && driver == "redis" {
		ctx.Logger().Warnw("auth driver redis is unavailable; falling back to memory", "error", err)
		stores, err = authstore.New("memory", authstore.Options{
			ResetCodeTTL: time.Duration(cfg.ResetCodeTTLMin) * time.Minute,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("init auth stores: %w", err)
	}
	mod := New(ctx.DB(), stores, *cfg,
		ctx.Config().HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier,
		ctx.Port("outbox"), ctx.Port("notification"), ctx.Port("encryption"),
		ctx.Port("tenant"), ctx.Port("mfa"), provisioner, ctx.Port("breach"), users,
		ctx.Port(contract.TenantInfoPortName), ctx.Port(contract.UserinfoPortName))
	if err := ctx.Provide(PortName, mod.Finalizer()); err != nil {
		return nil, fmt.Errorf("provide auth port: %w", err)
	}
	return mod, nil
}

// validateConfig 保留开通式注册必须允许公开注册的组合约束。
func validateConfig(cfg *Config, provisioningEnabled bool) error {
	if provisioningEnabled && !cfg.PublicRegistration {
		return errProvisioningRequiresPublicRegistration
	}
	return nil
}
