package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jimu/internal/app"
	adminmodule "jimu/internal/capabilities/admin"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/catalog"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/permission"
	"jimu/internal/capabilities/role"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/user"
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
	"user", "role", "permission", "tenant", "auth", "audit", "admin", "oauth",
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

	// 租户套餐/配额：定义在 tenant 能力，注入到创建用户/角色/API Key 的路径
	tenantMod := tenantmodule.New(container.DB, *cfg)

	// 能力开关：capabilities.enabled 为空表示全部启用（向后兼容）
	caps, err := catalog.Resolve(cfg.Capabilities.Enabled)
	if err != nil {
		_ = container.Stop(context.Background())
		return fmt.Errorf("resolve capabilities: %w", err)
	}

	// 全部装配的实例：键为能力名，仅覆盖 wiredCapabilities 名册；
	// 清单尾部基础设施能力只参与迁移，不构造实例（P1 收编）
	// 过渡实现（P0）：先构造再过滤；P1 引入显式 Deps 后改为按需构造
	all := map[string]contract.Module{
		"user":       user.New(container.DB, *cfg, container.Redis, container.Outbox),
		"auth":       authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease, container.Captcha, cfg.Captcha, container.Outbox, container.Notification, container.Cipher, container.BreachChecker, tenantMod.Quota()),
		"role":       role.New(container.DB, tenantMod.Quota()),
		"permission": permission.New(container.DB),
		"tenant":     tenantMod,
		"audit":      auditmodule.New(container.DB, cfg.Audit, container.Logger),
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
