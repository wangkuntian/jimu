package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jimu/internal/app"
	"jimu/internal/config"
	auditmodule "jimu/internal/modules/audit"
	authmodule "jimu/internal/modules/auth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	"jimu/internal/modules/user"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu Backend Framework API
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
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
	container, err := app.NewContainer(cfg)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	application, err := app.Bootstrap(
		container,
		user.New(container.DB),
		authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease),
		role.New(container.DB),
		permission.New(container.DB),
		auditmodule.New(container.DB, cfg.Audit, container.Logger),
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
