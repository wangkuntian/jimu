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
	"jimu/internal/platform/auth"
	"jimu/internal/platform/http/middleware"
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

	// 全部能力的实例：键为能力名，与 catalog 清单一一对应
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
	modules := make([]contract.Module, 0, len(caps))
	for _, d := range caps {
		module, ok := all[d.Name]
		if !ok {
			_ = container.Stop(context.Background())
			return fmt.Errorf("%w: %q", errCapabilityNotWired, d.Name)
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
