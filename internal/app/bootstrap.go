package app

import (
	"fmt"
	"time"

	"jimu/internal/contract"
	platformhttp "jimu/internal/platform/http"
	"jimu/internal/platform/observability"
)

func Bootstrap(container *Container, modules ...contract.Module) (*Application, error) {
	cfg := container.Config
	router := platformhttp.SetupRouter(container.Logger, cfg.HTTP, cfg.Server)
	if err := platformhttp.ConfigureTrustedProxies(router, cfg.HTTP.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	if cfg.HTTP.Mode != "release" {
		platformhttp.RegisterSwagger(router.Group("/swagger"))
	}

	for _, module := range modules {
		if provider, ok := module.(contract.HTTPMiddlewareProvider); ok {
			router.Use(provider.HTTPMiddleware()...)
		}
	}
	for _, module := range modules {
		module.RegisterHTTP(router)
		container.Logger.Info("module registered", "name", module.Name())
	}

	sqlDB, err := container.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	readiness := observability.NewReadiness(
		time.Duration(cfg.Management.ProbeTimeoutSec)*time.Second,
		observability.NewSQLChecker(sqlDB),
		observability.NewRedisChecker(container.Redis),
	)
	management := platformhttp.NewManagementServer(
		cfg.Management,
		platformhttp.HealthRouter(readiness, cfg.Management.EnablePprof),
	)
	public := platformhttp.NewServer(cfg.HTTP, router)

	components := []contract.Component{container}
	for _, module := range modules {
		if provider, ok := module.(contract.ComponentProvider); ok {
			components = append(components, provider.Components()...)
		}
	}
	components = append(components, management, public)
	return NewApplication(time.Duration(cfg.HTTP.ShutdownTimeoutSec)*time.Second, components...), nil
}
