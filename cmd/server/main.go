package main

import (
	"errors"
	"fmt"
	"os"

	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/captcha"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/feature"
	mfamodule "jimu/internal/capabilities/mfa"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/outbox"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	"jimu/internal/capabilities/search"
	"jimu/internal/capabilities/storage"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
	userapplication "jimu/internal/capabilities/user/application"
	userinfra "jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu 后端框架 API - 提供用户认证、权限管理、角色管理、系统监控等功能
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization

// version 版本号，通过 ldflags 注入：-ldflags "-X main.version=v0.1.0"
var version = "dev"

// errProvisioningRequiresPublicRegistration 开通式注册要求公开注册（组合根跨字段校验）
var errProvisioningRequiresPublicRegistration = errors.New("auth.provisioning.enabled requires auth.public_registration")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return assembly.Run(fullAssembly())
}

// fullAssembly 是本阶段的过渡形态：已搬迁的能力用真实 Wire，未搬迁的能力用内联
// Wire 闭包持有今天的构造代码；Task 3 逐个搬迁后，内联闭包会被 <name>.Wire 取代。
//
// 顺序即装配顺序：tenant/access 必须排在 user 之前（user 的角色分配/配额端口由二者
// 提供），captcha/mfa 必须排在 auth 之前。这是过渡期的接线顺序，与 catalog 的声明
// 顺序不同（catalog 把 user 放在最前，user 对 access/tenant 是软依赖）。
// 无 Module 实例的能力（outbox/search/breach）保留条目并给出空 Wire，使启用集与
// /capabilities 报告和今天逐值一致。
func fullAssembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "full",
		Version: version,
		Capabilities: []assembly.Capability{
			{Descriptor: tenantmodule.Descriptor, Wire: wireTenant},
			{Descriptor: accessmodule.Descriptor, Wire: wireAccess},
			{Descriptor: user.Descriptor, Wire: wireUser},
			{Descriptor: captcha.Descriptor, Wire: captcha.Wire}, // 已搬迁的试点能力
			{Descriptor: mfamodule.Descriptor, Wire: wireMFA},
			{Descriptor: authmodule.Descriptor, Wire: wireAuth},
			{Descriptor: passkeymodule.Descriptor, Wire: wirePasskey},
			{Descriptor: auditmodule.Descriptor, Wire: wireAudit},
			{Descriptor: consolemodule.Descriptor, Wire: wireConsole},
			{Descriptor: oauthmodule.Descriptor, Wire: wireOAuth},
			{Descriptor: apikey.Descriptor, Wire: wireAPIKey},
			{Descriptor: queue.Descriptor, Wire: wireQueue},
			{Descriptor: dataops.Descriptor, Wire: wireDataops},
			{Descriptor: feature.Descriptor, Wire: wireFeature},
			{Descriptor: uploadsec.Descriptor, Wire: wireUploadsec},
			{Descriptor: outbox.Descriptor, Wire: noModule},
			{Descriptor: search.Descriptor, Wire: noModule},
			{Descriptor: breach.Descriptor, Wire: noModule},
		},
	}
}

func wireTenant(ctx *assembly.Context) (contract.Module, error) {
	mod := tenantmodule.New(ctx.DB(), tenantProvisioningConfig(authConfig(ctx).Provisioning))
	if err := ctx.Provide("tenant", mod.Quota()); err != nil {
		return nil, err
	}
	if err := ctx.Provide("tenant.provisioner", mod.Provisioner()); err != nil {
		return nil, err
	}
	return mod, nil
}

func wireAccess(ctx *assembly.Context) (contract.Module, error) {
	mod := accessmodule.New(ctx.DB(), ctx.Port("tenant"))
	// access 是 user_roles 表所有者：把角色分配端口交给排在其后的 user（缺此 Provide 时
	// user 的 AssignRoles 会退化为 "role assignment is not configured"）。
	if err := ctx.Provide("access", mod.UserRoleAssigner()); err != nil {
		return nil, err
	}
	return mod, nil
}

func wireUser(ctx *assembly.Context) (contract.Module, error) {
	roles, _ := ctx.Port("access").(userapplication.UserRoleAssigner)
	quota, _ := ctx.Port("tenant").(userapplication.TenantQuota)
	outboxMod, _ := ctx.Port("outbox").(*outbox.Outbox)
	return user.New(ctx.DB(), *ctx.Config(), ctx.Redis(), outboxMod).WithRoles(roles).WithQuota(quota), nil
}

func wireMFA(ctx *assembly.Context) (contract.Module, error) {
	authCfg := authConfig(ctx)
	mod := mfamodule.New(ctx.DB(), mfamodule.Config{
		JWTSecret:         authCfg.JWTSecret,
		JWTPreviousSecret: authCfg.JWTPreviousSecret,
		Issuer:            authCfg.Issuer,
		AccessExpireMin:   authCfg.AccessExpireMin,
		RefreshExpireDay:  authCfg.RefreshExpireDay,
		TrustedDeviceDays: authCfg.TrustedDeviceDays,
	}, user.NewUserinfoSource(userinfra.NewMysqlRepository(ctx.DB())))
	if err := ctx.Provide("mfa", mod.Service()); err != nil {
		return nil, err
	}
	return mod, nil
}

func wireAuth(ctx *assembly.Context) (contract.Module, error) {
	authCfg := authConfig(ctx)
	if err := validateAuthConfig(authCfg); err != nil {
		return nil, err
	}
	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力未启用/未装配时端口取回
	// nil，登录/注册跳过验证码校验。
	captchaVerifier, _ := ctx.Port(captcha.PortName).(contract.CaptchaVerifier)
	mod := authmodule.New(ctx.DB(), ctx.Redis(), *authCfg,
		ctx.Config().HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier,
		ctx.Port("outbox"), ctx.Port("notification"), ctx.Port("encryption"),
		ctx.Port("tenant"), ctx.Port("mfa"), ctx.Port("tenant.provisioner"), ctx.Port("breach"))
	if err := ctx.Provide("auth", mod.Finalizer()); err != nil {
		return nil, err
	}
	return mod, nil
}

func wirePasskey(ctx *assembly.Context) (contract.Module, error) {
	finalizer, _ := ctx.Port("auth").(contract.LoginFinalizer)
	return passkeymodule.New(passkeymodule.Deps{
		DB:         ctx.DB(),
		Redis:      ctx.Redis(),
		AuthCfg:    *authConfig(ctx),
		Users:      user.NewUserinfoSource(userinfra.NewMysqlRepository(ctx.DB())),
		Finalizer:  finalizer,
		FailClosed: ctx.Config().HTTP.Mode == config.HTTPModeRelease,
	}), nil
}

func wireAudit(ctx *assembly.Context) (contract.Module, error) {
	return auditmodule.New(ctx.DB(), configSection(ctx, auditmodule.ConfigKey, func() *auditmodule.Config {
		return &auditmodule.Config{}
	}), ctx.Logger()), nil
}

func wireConsole(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	authCfg := authConfig(ctx)
	return consolemodule.New(cfg.Version, cfg.Environment, ctx.Redis(), ctx.DB(),
		auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay),
		ctx.EventBus(), middleware.IPAllowlist(cfg.Security.AdminIPAllowlist)), nil
}

func wireOAuth(ctx *assembly.Context) (contract.Module, error) {
	return oauthmodule.New(ctx.DB(), ctx.Redis(),
		configSection(ctx, oauthmodule.ConfigKey, func() *oauthmodule.Config { return &oauthmodule.Config{} }),
		*authConfig(ctx), ctx.HTTPClient()), nil
}

func wireAPIKey(ctx *assembly.Context) (contract.Module, error) {
	return apikey.New(ctx.DB(), ctx.Port("tenant")), nil
}

func wireQueue(ctx *assembly.Context) (contract.Module, error) {
	return queue.NewModule(ctx.DB(), ctx.Scheduler()), nil
}

func wireDataops(ctx *assembly.Context) (contract.Module, error) {
	return dataops.New(ctx.DB()), nil
}

func wireFeature(ctx *assembly.Context) (contract.Module, error) {
	return feature.New(ctx.DB()), nil
}

func wireUploadsec(ctx *assembly.Context) (contract.Module, error) {
	storageSvc, _ := ctx.Port("storage").(storage.Storage)
	scanner, _ := ctx.Port("uploadsec.scanner").(uploadsec.Scanner)
	return uploadsec.New(storageSvc, scanner), nil
}

// noModule 是「无 Module 实例」能力的 Wire：只携带声明（outbox/search/breach 参与
// 迁移与端口消费，但不注册模块）。
func noModule(*assembly.Context) (contract.Module, error) { return nil, nil }

// configSection 取能力配置段并解引用为值；段不存在（能力未启用）时用 zero 构造零值，
// 与旧装配惯例一致（未启用能力的配置段取零值即"不接线/默认行为"）。
func configSection[T any](ctx *assembly.Context, key string, zero func() *T) T {
	if cfg := assembly.MustSection[*T](ctx, key); cfg != nil {
		return *cfg
	}
	return *zero()
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}

// validateAuthConfig 组合根承担的 auth 段跨字段校验。
// 设计 §8 ¶2 不拆 auth 段，provisioning 与 public_registration 同段；但 provisioning
// 的语义归 tenant，P2.1 裁定把这条校验留在组合根（与 outbox.publisher 依赖 queue.type 同理）。
func validateAuthConfig(cfg *authmodule.Config) error {
	if cfg.Provisioning.Enabled && !cfg.PublicRegistration {
		return errProvisioningRequiresPublicRegistration
	}
	return nil
}

// tenantProvisioningConfig 把 auth 段的 provisioning 配置映射为 tenant 自有的输入视图。
// tenant 被 auth 依赖、不得 import auth，故两边类型独立，在此显式转换（P2.1 裁定）。
func tenantProvisioningConfig(p authmodule.ProvisioningConfig) tenantmodule.ProvisioningConfig {
	roles := make([]tenantmodule.ProvisionRoleTemplate, 0, len(p.Roles))
	for _, role := range p.Roles {
		perms := make([]tenantmodule.ProvisionPermission, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			perms = append(perms, tenantmodule.ProvisionPermission{Resource: perm.Resource, Action: perm.Action})
		}
		roles = append(roles, tenantmodule.ProvisionRoleTemplate{
			Name:        role.Name,
			Description: role.Description,
			Permissions: perms,
		})
	}
	return tenantmodule.ProvisioningConfig{Enabled: p.Enabled, OwnerRole: p.OwnerRole, Roles: roles}
}
