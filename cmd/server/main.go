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
	adminmodule "jimu/internal/capabilities/admin"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/captcha"
	"jimu/internal/capabilities/catalog"
	mfamodule "jimu/internal/capabilities/mfa"
	oauthmodule "jimu/internal/capabilities/oauth"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	tenantmodule "jimu/internal/capabilities/tenant"
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

// wiredCapabilities 是 main 装配的能力名册；必须是 catalog.Names() 的子集
// （清单尾部的基础设施能力只带迁移、尚无 Module 实例，不在名册中）。
// 单元测试（main_test.go）对账两者，run() 启动时按它过滤装配并自检实例映射。
var wiredCapabilities = []string{
	"user", "access", "tenant", "auth", "mfa", "passkey", "queue",
	"audit", "admin", "oauth", "captcha",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 注入元数据
	cfg.Version = version
	cfg.Environment = os.Getenv("APP_ENV")

	container, err := app.NewContainer(cfg)
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

	// 能力开关：capabilities.enabled 为空表示全部启用（向后兼容）
	caps, err := catalog.Resolve(cfg.Capabilities.Enabled)
	if err != nil {
		_ = container.Stop(context.Background())
		return fmt.Errorf("resolve capabilities: %w", err)
	}
	enabled := make(map[string]bool, len(caps))
	for _, d := range caps {
		enabled[d.Name] = true
	}

	// 租户套餐/配额/开通式注册：定义在 tenant 能力，注入到创建用户/角色/API Key 的路径
	tenantMod := tenantmodule.New(container.DB, *cfg)

	// 用户只读端口：mfa/passkey 经 contract.UserinfoSource 读取用户，不 import user/domain
	userinfoSource := user.NewUserinfoSource(userinfra.NewMysqlRepository(container.DB))
	// 验证码能力：公开 GET /api/v1/captcha，并经 contract.CaptchaVerifier 注入 auth
	captchaMod := captcha.New(container.Redis, time.Duration(cfg.Captcha.TTLMin)*time.Minute, cfg.Captcha.Enabled)
	// MFA 能力：TOTP + 可信设备，经 contract.MFAVerifier 注入 auth
	mfaMod := mfamodule.New(container.DB, cfg.Auth, userinfoSource)

	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力关闭时注入 nil，登录/注册跳过验证码校验；
	// mfa/tenant 是 auth 的硬依赖（Requires 保证启用），按实例注入即可。
	var captchaVerifier contract.CaptchaVerifier
	if enabled["captcha"] {
		captchaVerifier = captchaMod.Service()
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
	authMod := authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier, container.Outbox, container.Notification, container.Cipher,
		tenantMod.Quota(), mfaMod.Service(), tenantMod.Provisioner(), container.BreachChecker)
	// passkey 能力：WebAuthn 无密码登录，登录收尾经 contract.LoginFinalizer 委托 auth
	passkeyMod := passkeymodule.New(passkeymodule.Deps{
		DB:         container.DB,
		Redis:      container.Redis,
		AuthCfg:    cfg.Auth,
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
		"tenant":  tenantMod,
		"audit":   auditmodule.New(container.DB, cfg.Audit, container.Logger),
		"admin": adminmodule.New(cfg.Version, cfg.Environment, container.Redis, container.DB, middleware.IPAllowlist(cfg.Security.AdminIPAllowlist), container.Scheduler, container.Storage, container.UploadScanner, container.FeatureFlag, container.EventBus,
			auth.NewWithRotation(cfg.Auth.JWTSecret, cfg.Auth.JWTPreviousSecret, cfg.Auth.Issuer, cfg.Auth.AccessExpireMin, cfg.Auth.RefreshExpireDay),
			tenantMod.Quota()),
		"oauth": oauthmodule.New(container.DB, container.Redis, cfg.OAuth, cfg.Auth, container.HTTPClient),
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
