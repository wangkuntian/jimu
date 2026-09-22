package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jimu/internal/capability"
	"jimu/internal/contract"
	platformhttp "jimu/internal/kernel/http"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/observability"
	"jimu/internal/kernel/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	gormotel "gorm.io/plugin/opentelemetry/tracing"
)

// Bootstrap 在全部能力装配完成后接管生命周期：初始化观测、路由、事件与定时任务。
// capabilities/components/jobs 由装配驱动传入（app 不得 import 任何能力包）：
//   - modules 经 contract.Module 注册 HTTP/事件/任务；
//   - components 是能力贡献的 contract.Component（如 worker pool、WS Hub、gRPC server）；
//   - jobs 是能力贡献的定时任务定义（如 outbox_process、retention）。
func Bootstrap(container *Container, components []contract.Component, jobs []scheduler.Job, modules ...contract.Module) (*Application, error) {
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
	// 上一行的 count/names 是「已装配模块」集合；下面这行是组合根解析出的启用集
	// （可能含 outbox/search/breach 等无 Module 实例的能力），两者刻意分开打印。
	resolvedNames := make([]string, 0, len(container.Capabilities))
	for _, d := range container.Capabilities {
		resolvedNames = append(resolvedNames, d.Name)
	}
	container.Logger.Infow("capabilities resolved", "count", len(container.Capabilities), "names", strings.Join(resolvedNames, ","))
	// 软依赖缺失只降级、不阻断启用（设计 §6.4）：在 enabled 日志之后报告，
	// 被 fail-closed 拒绝的启用集不会留下降级噪音。
	for _, d := range capability.Degraded(container.Capabilities) {
		container.Logger.Warnw("capability degraded", "name", d.Capability, "missing", strings.Join(d.Missing, ","))
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
		platformhttp.HealthRouter(readiness, cfg.Management.EnablePprof, func(mux *http.ServeMux) {
			mux.HandleFunc("/capabilities", capabilitiesHandler(container.Capabilities))
		}),
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

		jobFns := map[string]scheduler.Job{}
		if container.DBCollector != nil {
			jobFns["metrics_collect"] = scheduler.Job{ID: "metrics_collect", Name: "Collect DB Metrics", Spec: "@every 15s", Run: func() {
				container.DBCollector.Collect()
				observability.CollectRuntime()
			}}
		}
		// 能力贡献的定时任务（outbox_process、cleanup、retention 等）
		for _, job := range jobs {
			if _, dup := jobFns[job.ID]; dup {
				return nil, fmt.Errorf("duplicate scheduled job id %q contributed by a capability", job.ID)
			}
			jobFns[job.ID] = job
		}

		// 从 store 恢复持久化任务，跳过已恢复 id，防双注册
		restored, err := container.Scheduler.RestoreFromStore(context.Background(), func(id string) func() {
			if def, ok := jobFns[id]; ok {
				return def.Run
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
			if err := container.Scheduler.AddNamedFunc(id, def.Name, def.Spec, def.Run); err != nil {
				container.Logger.Errorw("register job failed", "id", id, "error", err.Error())
			}
		}
	}

	lifecycle := []contract.Component{container}
	lifecycle = append(lifecycle, components...)
	if container.Scheduler != nil {
		lifecycle = append(lifecycle, container.Scheduler)
	}
	for _, module := range modules {
		if provider, ok := module.(contract.ComponentProvider); ok {
			lifecycle = append(lifecycle, provider.Components()...)
		}
	}
	lifecycle = append(lifecycle, management, public)
	return NewApplication(time.Duration(cfg.HTTP.ShutdownTimeoutSec)*time.Second, lifecycle...), nil
}

// capabilitiesResponse 是 /capabilities 的响应体；字段声明顺序即 JSON 键顺序。
type capabilitiesResponse struct {
	Enabled  []string                 `json:"enabled"`
	Degraded []capability.Degradation `json:"degraded"`
}

// capabilitiesHandler 输出最终启用清单与降级项（设计 §6.4）。管理端口只读、不鉴权。
func capabilitiesHandler(caps []contract.Descriptor) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		names := make([]string, 0, len(caps))
		for _, d := range caps {
			names = append(names, d.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(capabilitiesResponse{
			Enabled:  names,
			Degraded: capability.Degraded(caps),
		})
	}
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
	// 返回空链的提供者视为显式让位（如 auth 已在时 apikey 不接管），不构成第二个提供者。
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
		if len(chain) == 0 {
			continue
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
				return fmt.Errorf("capability %q declares MountProtected but no enabled capability provides protected middleware; enable the capability that provides it (currently \"auth\" or \"apikey\")", desc.Name)
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
