package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jimu/internal/app"
	"jimu/internal/config"
	adminmodule "jimu/internal/modules/admin"
	auditmodule "jimu/internal/modules/audit"
	authmodule "jimu/internal/modules/auth"
	oauthmodule "jimu/internal/modules/oauth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	"jimu/internal/modules/user"
	"jimu/internal/platform/auth"
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
		container.Logger.Info("config file changed, applying runtime settings", "level", newCfg.Log.Level)
		if err := container.Logger.SetLevel(newCfg.Log.Level); err != nil {
			container.Logger.Error("apply new log level failed", "error", err.Error())
			return err
		}
		return nil
	}); err != nil {
		container.Logger.Warn("config file watch disabled", "error", err.Error())
	}

	application, err := app.Bootstrap(
		container,
		user.New(container.DB, *cfg, container.Redis, container.Outbox),
		authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease, container.Captcha, cfg.Captcha, container.Outbox, container.Notification, container.Cipher),
		role.New(container.DB),
		permission.New(container.DB),
		auditmodule.New(container.DB, cfg.Audit, container.Logger),
		adminmodule.New(cfg.Version, cfg.Environment, container.Redis, container.DB, container.Scheduler, container.Storage, container.FeatureFlag, container.EventBus,
			auth.NewWithRotation(cfg.Auth.JWTSecret, cfg.Auth.JWTPreviousSecret, cfg.Auth.Issuer, cfg.Auth.AccessExpireMin, cfg.Auth.RefreshExpireDay)),
		oauthmodule.New(container.DB, container.Redis, cfg.OAuth, cfg.Auth, container.HTTPClient),
	)
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
