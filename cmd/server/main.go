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

	application, err := app.Bootstrap(
		container,
		user.New(container.DB, *cfg, container.Redis, container.Outbox),
		authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease, container.Captcha, cfg.Captcha, container.Outbox),
		role.New(container.DB),
		permission.New(container.DB),
		auditmodule.New(container.DB, cfg.Audit, container.Logger),
		adminmodule.New(cfg.Version, cfg.Environment, container.Redis, container.DB, container.Scheduler, container.Storage, container.FeatureFlag, container.EventBus),
		oauthmodule.New(container.DB, container.Redis, cfg.OAuth, cfg.Auth),
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
