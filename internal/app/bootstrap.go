package app

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/contract"
	"jimu/internal/platform/db"
	platformhttp "jimu/internal/platform/http"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/observability"

	"github.com/gin-gonic/gin"
)

func Bootstrap(container *Container, modules ...contract.Module) (*Application, error) {
	cfg := container.Config

	// 初始化 OpenTelemetry 追踪
	tp, err := observability.InitTracing(context.Background(), cfg.OTEL)
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}
	container.TracerProvider = tp

	router := platformhttp.SetupRouter(container.Logger, cfg.HTTP, cfg.Server, cfg.Security, cfg.OTEL)
	if err := platformhttp.ConfigureTrustedProxies(router, cfg.HTTP.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	if cfg.HTTP.Mode != "release" {
		platformhttp.RegisterSwagger(router.Group("/swagger"))
	}

	if err := registerHTTP(router, container.Logger, modules...); err != nil {
		return nil, err
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

	// 注册各模块的事件处理器（在定时任务之前，确保事件订阅就绪）
	for _, module := range modules {
		module.RegisterEvents(container.EventBus)
		container.Logger.Info("module events registered", "name", module.Name())
	}

	// 注册全局事件处理器：将领域事件桥接到通知系统
	if container.Notification != nil {
		container.EventBus.Subscribe(contract.UserCreatedEmailNotification, func(payload interface{}) {
			if msg, ok := payload.(notification.Message); ok {
				if err := container.Notification.Dispatch(context.Background(), msg); err != nil {
					container.Logger.Error("notification dispatch failed", "error", err.Error())
				}
			}
		})
	}

	// 注册各模块的定时任务
	if container.JobRegistry != nil {
		for _, module := range modules {
			module.RegisterJobs(container.JobRegistry)
			container.Logger.Info("module jobs registered", "name", module.Name())
		}

		// 注册 Outbox 定时处理（每 10 秒处理一次待发布事件）
		if container.Outbox != nil {
			_ = container.Scheduler.AddNamedFunc("outbox_process", "Process Outbox Events", "@every 10s", func() {
				n, err := container.Outbox.Process(context.Background(), 100)
				if err != nil {
					container.Logger.Error("outbox process error", "error", err.Error())
				} else if n > 0 {
					container.Logger.Debug("outbox processed", "count", n)
				}
			})
		}

		// 注册 DB 指标收集（每 15 秒收集一次）
		if container.DBCollector != nil {
			_ = container.Scheduler.AddNamedFunc("metrics_collect", "Collect DB Metrics", "@every 15s", func() {
				container.DBCollector.Collect()
				observability.CollectRuntime()
			})
		}

		// 注册 WebSocket Hub 运行
		if container.WebSocketHub != nil {
			go container.WebSocketHub.Run(context.Background())
		}

		// 注册数据清理 Job（每天凌晨 3 点清理超过 90 天的软删除数据）
		cleanupSvc := db.NewCleanupService(container.DB, db.DefaultCleanupConfig())
		_ = container.Scheduler.AddNamedFunc("cleanup", "Data Cleanup", "0 3 * * *", func() {
			results, err := cleanupSvc.Run(context.Background())
			if err != nil {
				container.Logger.Error("cleanup job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					container.Logger.Info("cleanup completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		})
	}

	components := []contract.Component{container}
	if container.Scheduler != nil {
		components = append(components, container.Scheduler)
	}
	for _, module := range modules {
		if provider, ok := module.(contract.ComponentProvider); ok {
			components = append(components, provider.Components()...)
		}
	}
	components = append(components, management, public)
	return NewApplication(time.Duration(cfg.HTTP.ShutdownTimeoutSec)*time.Second, components...), nil
}

type moduleLogger interface {
	Info(args ...interface{})
}

type registerRouter interface {
	contract.Router
	Use(...gin.HandlerFunc) gin.IRoutes
}

func registerHTTP(router registerRouter, log moduleLogger, modules ...contract.Module) error {
	for _, module := range modules {
		if provider, ok := module.(contract.HTTPMiddlewareProvider); ok {
			router.Use(provider.HTTPMiddleware()...)
		}
	}
	var protected []gin.HandlerFunc
	for _, module := range modules {
		if provider, ok := module.(contract.ProtectedHTTPMiddlewareProvider); ok {
			var err error
			protected, err = provider.ProtectedHTTPMiddleware()
			if err != nil {
				return fmt.Errorf("configure protected middleware: %w", err)
			}
			break
		}
	}
	for _, module := range modules {
		if len(protected) > 0 && module.Name() != "auth" && module.Name() != "oauth" {
			group := router.Group("", protected...)
			module.RegisterHTTP(group)
			if log != nil {
				log.Info("module registered", "name", module.Name())
			}
			continue
		}
		module.RegisterHTTP(router)
		if log != nil {
			log.Info("module registered", "name", module.Name())
		}
	}
	return nil
}
