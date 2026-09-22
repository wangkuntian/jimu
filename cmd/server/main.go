package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jimu/internal/app"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/captcha"
	"jimu/internal/capabilities/catalog"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/feature"
	mfamodule "jimu/internal/capabilities/mfa"
	oauthmodule "jimu/internal/capabilities/oauth"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
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

// errCapabilityNotWired 表示 catalog 声明了能力但 main 未提供实例（开发期配置错误）
var errCapabilityNotWired = errors.New("capability declared in catalog but not wired in main")

// errCapabilityWiringMismatch 表示装配映射与声明名册的规模不一致（开发期配置错误）
var errCapabilityWiringMismatch = errors.New("capability wiring mismatch")

// errCapabilityNoInstance 表示声明名册中的能力在装配映射里没有实例（开发期配置错误）
var errCapabilityNoInstance = errors.New("declared capability has no instance")

// errProvisioningRequiresPublicRegistration 开通式注册要求公开注册（组合根跨字段校验）
var errProvisioningRequiresPublicRegistration = errors.New("auth.provisioning.enabled requires auth.public_registration")

// wiredCapabilities 是 main 装配的能力名册；必须是 catalog.Names() 的子集
// （清单尾部的基础设施能力只带迁移、尚无 Module 实例，不在名册中）。
// 单元测试（main_test.go）对账两者，run() 启动时按它过滤装配并自检实例映射。
var wiredCapabilities = []string{
	"user", "access", "tenant", "auth", "mfa", "passkey", "queue", "apikey", "dataops",
	"audit", "console", "feature", "uploadsec", "oauth", "captcha",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, sections, err := config.LoadWithSections()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 注入元数据
	cfg.Version = version
	cfg.Environment = os.Getenv("APP_ENV")

	// 能力开关：capabilities.enabled 为空表示全部启用（向后兼容）。
	// 在构建容器前解析，因为能力配置段按启用集加载（设计 §8）。
	caps, err := catalog.Resolve(cfg.Capabilities.Enabled)
	if err != nil {
		return fmt.Errorf("resolve capabilities: %w", err)
	}
	enabled := make(map[string]bool, len(caps))
	for _, d := range caps {
		enabled[d.Name] = true
	}

	// 能力配置段：按启用集解码 → 默认值 → 校验（prod 下追加加严校验）。
	// 未启用的能力不在 caps 内，其配置段既不出现也不校验（设计 §8）。
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, os.Getenv("APP_ENV"))
	if err != nil {
		return fmt.Errorf("load capability configs: %w", err)
	}
	// auth 段由 auth 能力声明：未启用 auth 时该段不加载，实例仍按旧装配惯例构造但不挂路由。
	authCfg, _ := app.SectionOf[*authmodule.Config](capCfgs, authmodule.ConfigKey)
	if authCfg == nil {
		authCfg = &authmodule.Config{}
	}
	if err := validateAuthConfig(authCfg); err != nil {
		return err
	}

	container, err := app.NewContainer(cfg, sections, capCfgs, caps, enabled)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	// 配置文件热更新：仅应用运行时安全项（log.level）。
	// 结构类配置（DB/Redis 连接池、监听端口等）变更需重启进程生效。
	if err := config.Watch(func(newCfg *config.Config) error {
		container.Logger.Infow("config file changed, applying runtime settings", "level", newCfg.Log.Level)
		if err := container.Logger.SetLevel(newCfg.Log.Level); err != nil {
			container.Logger.Errorw("apply new log level failed", "error", err.Error())
			return err
		}
		return nil
	}); err != nil {
		container.Logger.Warnw("config file watch disabled", "error", err.Error())
	}

	// 租户套餐/配额/开通式注册：定义在 tenant 能力，注入到创建用户/角色/API Key 的路径。
	// provisioning 配置由组合根从 auth 段构造（tenant 不得 import auth，见 tenantProvisioningConfig）。
	tenantMod := tenantmodule.New(container.DB, tenantProvisioningConfig(authCfg.Provisioning))

	// 用户只读端口：mfa/passkey 经 contract.UserinfoSource 读取用户，不 import user/domain
	userinfoSource := user.NewUserinfoSource(userinfra.NewMysqlRepository(container.DB))
	// 验证码能力：公开 GET /api/v1/captcha，并经 contract.CaptchaVerifier 注入 auth。
	// 配置段由能力声明并已按启用集加载（未启用则该段不出现，取零值即关闭）。
	var captchaCfg captcha.Config
	if c, ok := app.SectionOf[*captcha.Config](capCfgs, captcha.ConfigKey); ok {
		captchaCfg = *c
	}
	captchaMod := captcha.New(container.Redis, time.Duration(captchaCfg.TTLMin)*time.Minute, captchaCfg.Enabled)
	// MFA 能力：TOTP + 可信设备，经 contract.MFAVerifier 注入 auth。
	// mfa 的 Requires 只有 user，不得 import auth，故 JWT/可信设备参数经装配期配置传入。
	mfaMod := mfamodule.New(container.DB, mfamodule.Config{
		JWTSecret:         authCfg.JWTSecret,
		JWTPreviousSecret: authCfg.JWTPreviousSecret,
		Issuer:            authCfg.Issuer,
		AccessExpireMin:   authCfg.AccessExpireMin,
		RefreshExpireDay:  authCfg.RefreshExpireDay,
		TrustedDeviceDays: authCfg.TrustedDeviceDays,
	}, userinfoSource)

	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力关闭时注入 nil，登录/注册跳过验证码校验；
	// mfa/tenant 是 auth 的硬依赖（Requires 保证启用），按实例注入即可。
	var captchaVerifier contract.CaptchaVerifier
	if enabled["captcha"] {
		captchaVerifier = captchaMod.Service()
	}

	// OAuth 能力配置段：由能力声明并按启用集加载
	var oauthCfg oauthmodule.Config
	if c, ok := app.SectionOf[*oauthmodule.Config](capCfgs, oauthmodule.ConfigKey); ok {
		oauthCfg = *c
	}

	// 审计能力配置段：由能力声明并按启用集加载
	var auditCfg auditmodule.Config
	if c, ok := app.SectionOf[*auditmodule.Config](capCfgs, auditmodule.ConfigKey); ok {
		auditCfg = *c
	}

	// access 能力：角色/权限/用户角色分配（user_roles 表所有者），供 user 管理面注入
	accessMod := accessmodule.New(container.DB, tenantMod.Quota())

	// user 能力的装配期端口注入：access 提供角色分配、tenant 提供配额
	userMod := user.New(container.DB, *cfg, container.Redis, container.Outbox).
		WithRoles(accessMod.UserRoleAssigner()).
		WithQuota(tenantMod.Quota())

	// 全部装配的实例：键为能力名，仅覆盖 wiredCapabilities 名册；
	// 清单尾部基础设施能力只参与迁移，不构造实例（P1 收编）
	// 过渡实现（P0）：先构造再过滤；P1 引入显式 Deps 后改为按需构造
	// auth 能力：会话/凭证/登录历史/密码历史；经端口消费 mfa/tenant/breach/captcha
	authMod := authmodule.New(container.DB, container.Redis, *authCfg, cfg.HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier, container.Outbox, container.Notification, container.Cipher,
		tenantMod.Quota(), mfaMod.Service(), tenantMod.Provisioner(), container.BreachChecker)
	// passkey 能力：WebAuthn 无密码登录，登录收尾经 contract.LoginFinalizer 委托 auth
	passkeyMod := passkeymodule.New(passkeymodule.Deps{
		DB:         container.DB,
		Redis:      container.Redis,
		AuthCfg:    *authCfg,
		Users:      userinfoSource,
		Finalizer:  authMod.Finalizer(),
		FailClosed: cfg.HTTP.Mode == config.HTTPModeRelease,
	})

	all := map[string]contract.Module{
		"user":    userMod,
		"auth":    authMod,
		"mfa":     mfaMod,
		"passkey": passkeyMod,
		"captcha": captchaMod,
		"access":  accessMod,
		"queue":   queue.NewModule(container.DB, container.Scheduler),
		"apikey":  apikey.New(container.DB, tenantMod.Quota()),
		"dataops": dataops.New(container.DB),
		"tenant":  tenantMod,
		"audit":   auditmodule.New(container.DB, auditCfg, container.Logger),
		"console": consolemodule.New(cfg.Version, cfg.Environment, container.Redis, container.DB,
			auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay),
			container.EventBus, middleware.IPAllowlist(cfg.Security.AdminIPAllowlist)),
		"feature":   feature.New(container.DB),
		"uploadsec": uploadsec.New(container.Storage, container.UploadScanner),
		"oauth":     oauthmodule.New(container.DB, container.Redis, oauthCfg, *authCfg, container.HTTPClient),
	}
	if len(all) != len(wiredCapabilities) {
		_ = container.Stop(context.Background())
		return fmt.Errorf("%w: %d instances for %d declared names", errCapabilityWiringMismatch, len(all), len(wiredCapabilities))
	}
	for _, name := range wiredCapabilities {
		if _, ok := all[name]; !ok {
			_ = container.Stop(context.Background())
			return fmt.Errorf("%w: %q", errCapabilityNoInstance, name)
		}
	}
	// 装配过滤：catalog.Resolve 可能返回暂无 Module 实例的基础设施能力
	// （仅参与迁移执行），它们不进入 Bootstrap。
	wired := make(map[string]bool, len(wiredCapabilities))
	for _, name := range wiredCapabilities {
		wired[name] = true
	}
	modules := make([]contract.Module, 0, len(caps))
	for _, d := range caps {
		module, ok := all[d.Name]
		if !ok {
			if wired[d.Name] {
				_ = container.Stop(context.Background())
				return fmt.Errorf("%w: %q", errCapabilityNotWired, d.Name)
			}
			continue
		}
		modules = append(modules, module)
	}

	application, err := app.Bootstrap(container, modules...)
	if err != nil {
		_ = container.Stop(context.Background())
		return fmt.Errorf("bootstrap application: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := application.Run(ctx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}
	return nil
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
