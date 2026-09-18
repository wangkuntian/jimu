package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/db"
	platformhttp "jimu/internal/platform/http"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/observability"
	"jimu/internal/platform/outbox"
	"jimu/internal/platform/queue"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	gormotel "gorm.io/plugin/opentelemetry/tracing"
)

// outboxTypeConverters 按事件类型将 outbox 内层 Payload 还原为强类型事件。
// 返回 error：载荷与事件类型不匹配时拒绝发布，避免零值事件被静默发出。
var outboxTypeConverters = map[string]func(json.RawMessage) (interface{}, error){
	contract.EventUserCreated: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserCreatedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserUpdated: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserUpdatedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserDeleted: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserDeletedEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
	contract.EventUserLoggedIn: func(p json.RawMessage) (interface{}, error) {
		var e contract.UserLoggedInEvent
		if err := json.Unmarshal(p, &e); err != nil {
			return nil, err
		}
		return e, nil
	},
}

// bridgeFn 反序列化 outbox 载荷并发布强类型事件到全局业务主题（裸主题）
func bridgeFn(c *Container) queue.WorkerFunc {
	return func(ctx context.Context, payload string) error {
		var evt outbox.EventPayload
		if err := json.Unmarshal([]byte(payload), &evt); err != nil {
			return fmt.Errorf("unmarshal outbox event: %w", err)
		}
		conv, ok := outboxTypeConverters[evt.EventType]
		if !ok {
			return fmt.Errorf("no converter for outbox event type: %s", evt.EventType)
		}
		strong, err := conv(evt.Payload)
		if err != nil {
			return fmt.Errorf("convert outbox event %s: %w", evt.EventType, err)
		}
		c.EventBus.Publish(evt.EventType, strong)
		return nil
	}
}

// registerOutboxWorkers 注册 MQ 消费端的 outbox 桥接 worker
func registerOutboxWorkers(c *Container) {
	for eventType := range outboxTypeConverters {
		eventType := eventType
		queue.RegisterWorker("outbox:"+eventType, bridgeFn(c))
	}
}

// registerEventBusBridge 订阅全局总线 outbox:* 主题，转强类型后发布到裸业务主题（event_bus 模式）
func registerEventBusBridge(c *Container) {
	for eventType := range outboxTypeConverters {
		eventType := eventType
		c.EventBus.Subscribe("outbox:"+eventType, func(payload interface{}) {
			evt, ok := payload.(outbox.EventPayload)
			if !ok {
				c.Logger.Error("outbox bridge: unexpected payload type")
				return
			}
			conv, ok := outboxTypeConverters[evt.EventType]
			if !ok {
				c.Logger.Errorw("outbox bridge: unknown event type", "type", evt.EventType)
				return
			}
			strong, err := conv(evt.Payload)
			if err != nil {
				c.Logger.Errorw("outbox bridge: convert event failed", "type", evt.EventType, "error", err.Error())
				return
			}
			c.EventBus.Publish(evt.EventType, strong)
		})
	}
}

func Bootstrap(container *Container, modules ...contract.Module) (*Application, error) {
	cfg := container.Config

	// 初始化 OpenTelemetry 追踪
	tp, err := observability.InitTracing(context.Background(), cfg.OTEL)
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}
	container.TracerProvider = tp

	// 指标推送（OTLP/gRPC → OpenObserve）：基于 Prometheus 默认 registry，
	// 现有 promauto 采集逻辑不变，/metrics 端点继续可用。
	if cfg.OTEL.Enabled && cfg.OTEL.MetricsEnabled {
		reg, ok := prometheus.DefaultRegisterer.(*prometheus.Registry)
		if !ok {
			container.Logger.Errorw("openobserve metrics pusher init failed", "error", "default registerer type mismatch")
		} else {
			pusher, err := observability.NewMetricsPusher(context.Background(), cfg.OTEL, reg)
			if err != nil {
				container.Logger.Errorw("openobserve metrics pusher init failed", "error", err.Error())
			} else {
				pusher.Start()
				container.MetricsPusher = pusher
			}
		}
	}

	// 启用 OTel 时插桩 DB/Redis，捕获查询子 span。
	// 必须在 InitTracing 之后：插件创建时固化全局 TracerProvider，
	// 提前实例化会绑定 NoOp provider 导致 span 永不产生。
	if cfg.OTEL.Enabled {
		if err := container.DB.Use(gormotel.NewPlugin()); err != nil {
			return nil, fmt.Errorf("init gorm otel plugin: %w", err)
		}
		if err := redisotel.InstrumentTracing(container.Redis); err != nil {
			return nil, fmt.Errorf("init redis otel tracing: %w", err)
		}
	}

	router := platformhttp.SetupRouter(container.Logger, cfg.HTTP, cfg.Server, cfg.Security, cfg.OTEL, container.Reporter)
	if err := platformhttp.ConfigureTrustedProxies(router, cfg.HTTP.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	if cfg.HTTP.Mode != "release" {
		platformhttp.RegisterSwagger(router.Group("/swagger"))
	}

	// 租户维度限流（Redis 滑动窗口）：挂在受保护中间件之后，平台级视角跳过
	var extraProtected []gin.HandlerFunc
	if container.Redis != nil && cfg.RateLimit.Tenant.Enabled && cfg.RateLimit.Tenant.Limit > 0 {
		extraProtected = append(extraProtected, middleware.TenantRateLimitMiddleware(
			container.Redis,
			cfg.RateLimit.Tenant.Limit,
			time.Duration(cfg.RateLimit.Tenant.WindowSec)*time.Second,
		))
	}
	// 幂等中间件：挂在认证/租户注入之后，键按租户+用户+方法+路径绑定
	if container.Redis != nil && cfg.Security.IdempotencyEnabled {
		extraProtected = append(extraProtected, middleware.IdempotencyMiddleware(
			container.Redis,
			time.Duration(cfg.Security.IdempotencyTTLSec)*time.Second,
		))
	}
	if err := registerHTTP(router, container.Logger, extraProtected, modules...); err != nil {
		return nil, err
	}
	// 在 registerHTTP 成功之后打印：被 fail-closed 拒绝的启用集不应留下 "enabled" 日志。
	names := make([]string, 0, len(modules))
	for _, module := range modules {
		names = append(names, contract.Describe(module).Name)
	}
	container.Logger.Infow("capabilities enabled", "count", len(names), "names", strings.Join(names, ","))

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
	public, err := platformhttp.NewServer(cfg.HTTP, router)
	if err != nil {
		return nil, fmt.Errorf("create http server: %w", err)
	}

	// 注册各模块的事件处理器（在定时任务之前，确保事件订阅就绪）
	for _, module := range modules {
		module.RegisterEvents(container.EventBus)
		container.Logger.Infow("module events registered", "name", module.Name())
	}

	// 注册全局事件处理器：将领域事件桥接到通知系统
	if container.Notification != nil {
		container.EventBus.Subscribe(contract.UserCreatedEmailNotification, func(payload interface{}) {
			if msg, ok := payload.(notification.Message); ok {
				if err := container.Notification.Dispatch(context.Background(), msg); err != nil {
					container.Logger.Errorw("notification dispatch failed", "error", err.Error())
				}
			}
		})
	}

	// 注册配置热更新处理器
	container.EventBus.Subscribe("config.updated", func(payload interface{}) {
		m, ok := payload.(map[string]string)
		if !ok {
			container.Logger.Warn("config.updated payload type mismatch")
			return
		}
		switch m["key"] {
		case "log_level":
			if err := container.Logger.SetLevel(m["value"]); err != nil {
				container.Logger.Errorw("apply config.updated log_level failed", "error", err.Error())
				return
			}
		}
		container.Logger.Infow("config updated applied", "key", m["key"], "value", m["value"])
	})

	// 注册各模块的定时任务
	if container.JobRegistry != nil {
		for _, module := range modules {
			module.RegisterJobs(container.JobRegistry)
			container.Logger.Infow("module jobs registered", "name", module.Name())
		}

		type jobDef struct {
			name string
			spec string
			fn   func()
		}
		jobFns := map[string]jobDef{}
		if container.Outbox != nil {
			jobFns["outbox_process"] = jobDef{name: "Process Outbox Events", spec: "@every 10s", fn: func() {
				n, err := container.Outbox.Process(context.Background(), 100)
				if err != nil {
					container.Logger.Errorw("outbox process error", "error", err.Error())
				} else if n > 0 {
					container.Logger.Debugw("outbox processed", "count", n)
				}
			}}
		}
		if container.DBCollector != nil {
			jobFns["metrics_collect"] = jobDef{name: "Collect DB Metrics", spec: "@every 15s", fn: func() {
				container.DBCollector.Collect()
				observability.CollectRuntime()
			}}
		}
		if container.DB != nil {
			cleanupSvc := db.NewCleanupService(container.DB, db.DefaultCleanupConfig())
			jobFns["cleanup"] = jobDef{name: "Data Cleanup", spec: "0 3 * * *", fn: func() {
				results, err := cleanupSvc.Run(context.Background())
				if err != nil {
					container.Logger.Errorw("cleanup job failed", "error", err.Error())
					return
				}
				for _, r := range results {
					if r.Deleted > 0 {
						container.Logger.Infow("cleanup completed", "table", r.Table, "deleted", r.Deleted)
					}
				}
			}}
		}

		if container.DB != nil && cfg.Retention.Enabled {
			retentionSvc := db.NewRetentionService(container.DB, cfg.Retention)
			spec := cfg.Retention.Cron
			if spec == "" {
				spec = "30 3 * * *"
			}
			jobFns["retention"] = jobDef{name: "History Retention", spec: spec, fn: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				results, err := retentionSvc.Run(ctx)
				if err != nil {
					container.Logger.Errorw("retention job failed", "error", err.Error())
					return
				}
				for _, r := range results {
					if r.Deleted > 0 {
						container.Logger.Infow("retention completed", "table", r.Table, "deleted", r.Deleted)
					}
				}
			}}
		}

		// 注册 WebSocket Hub 运行
		if container.WebSocketHub != nil {
			go container.WebSocketHub.Run(context.Background())
		}

		// 从 store 恢复持久化任务，跳过已恢复 id，防双注册
		restored, err := container.Scheduler.RestoreFromStore(context.Background(), func(id string) func() {
			if def, ok := jobFns[id]; ok {
				return def.fn
			}
			return nil
		})
		if err != nil {
			container.Logger.Errorw("restore scheduled jobs failed", "error", err.Error())
		}
		restoredSet := make(map[string]struct{}, len(restored))
		for _, id := range restored {
			restoredSet[id] = struct{}{}
		}
		for id, def := range jobFns {
			if _, ok := restoredSet[id]; ok {
				continue
			}
			if err := container.Scheduler.AddNamedFunc(id, def.name, def.spec, def.fn); err != nil {
				container.Logger.Errorw("register job failed", "id", id, "error", err.Error())
			}
		}
	}

	// 接线 outbox 事件消费：MQ 模式注册 worker 并启动 WorkerPool；event_bus 模式注册全局总线桥接器
	switch cfg.Outbox.Publisher {
	case config.OutboxPublisherMQ:
		registerOutboxWorkers(container)
	case config.OutboxPublisherEventBus:
		registerEventBusBridge(container)
	}

	components := []contract.Component{container}
	if container.WorkerPool != nil {
		components = append(components, workerPoolComponent{pool: container.WorkerPool})
	}
	if container.Scheduler != nil {
		components = append(components, container.Scheduler)
	}
	for _, module := range modules {
		if provider, ok := module.(contract.ComponentProvider); ok {
			components = append(components, provider.Components()...)
		}
	}
	if cfg.GRPC.Enabled && container.GRPCServer != nil {
		components = append(components, container.GRPCServer)
	}
	components = append(components, management, public)
	return NewApplication(time.Duration(cfg.HTTP.ShutdownTimeoutSec)*time.Second, components...), nil
}

// workerPoolComponent 包装 WorkerPool，实现 contract.Component 以纳入应用生命周期
type workerPoolComponent struct {
	pool *queue.WorkerPool
}

func (w workerPoolComponent) Start(context.Context) error {
	w.pool.Start()
	return nil
}

func (w workerPoolComponent) Stop(context.Context) error {
	w.pool.Stop()
	return nil
}

type registerRouter interface {
	contract.Router
	Use(...gin.HandlerFunc) gin.IRoutes
}

func registerHTTP(router registerRouter, log *logger.Logger, extraProtected []gin.HandlerFunc, modules ...contract.Module) error {
	// 全局中间件：所有能力声明的前置中间件（如审计写入）
	for _, module := range modules {
		if provider, ok := module.(contract.HTTPMiddlewareProvider); ok {
			router.Use(provider.HTTPMiddleware()...)
		}
	}
	// 受保护中间件：必须恰好由一个能力提供。多个提供者时无法仅凭 catalog 顺序
	// 判定认证/租户注入/限流链的组合语义，因此拒绝启动而不是"首个提供者生效"。
	var protected []gin.HandlerFunc
	providers := make([]string, 0, 1)
	for _, module := range modules {
		provider, ok := module.(contract.ProtectedHTTPMiddlewareProvider)
		if !ok {
			continue
		}
		chain, err := provider.ProtectedHTTPMiddleware()
		if err != nil {
			return fmt.Errorf("configure protected middleware: %w", err)
		}
		providers = append(providers, contract.Describe(module).Name)
		protected = append(protected, chain...)
	}
	if len(providers) > 1 {
		return fmt.Errorf("multiple capabilities provide protected middleware (%s); an explicit ordering rule is required", strings.Join(providers, ", "))
	}
	// extraProtected 只含租户限流/幂等等补充中间件，不能替代认证与租户注入，
	// 因此「是否存在受保护中间件」必须在追加 extraProtected 之前判定。
	hasProtectedMiddleware := len(protected) > 0
	// 追加外部注入的受保护中间件（如租户维度限流），顺序在认证/租户注入之后
	protected = append(protected, extraProtected...)
	for _, module := range modules {
		desc := contract.Describe(module)
		if desc.Normalized() == contract.MountProtected {
			if !hasProtectedMiddleware {
				return fmt.Errorf("capability %q declares MountProtected but no enabled capability provides protected middleware; enable the capability that provides it (currently \"auth\")", desc.Name)
			}
			module.RegisterHTTP(router.Group("", protected...))
		} else {
			module.RegisterHTTP(router)
		}
		if log != nil {
			log.Infow("capability registered", "name", desc.Name, "mount", string(desc.Normalized()))
		}
	}
	return nil
}
