package app

import (
	"context"
	"errors"
	"slices"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/event"
	"jimu/internal/platform/httpclient"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/metrics"
	"jimu/internal/platform/reporter"
	"jimu/internal/platform/scheduler"
)

// Container 只持有内核/基础设施件：能力件一律由各能力的 Wire 经装配上下文构造，
// 并经端口注册表互相消费（设计 §6.3）。此处不得再 import 任何能力包。
type Container struct {
	Config *config.Config
	// Sections 按 YAML 点分键解码能力配置段（能力配置由能力自身声明，设计 §8）
	Sections config.SectionDecoder
	// CapabilityConfigs 已按启用集解码并校验的能力配置段（P2.1 起，未启用的段不出现）
	CapabilityConfigs *CapabilityConfigs
	// Enabled 已启用能力名集合（含依赖闭包）
	Enabled map[string]bool
	// Capabilities 已解析启用集的能力描述符（含依赖闭包，按清单顺序）
	Capabilities []contract.Descriptor
	// DB 是 DB-enabled 构建的 *gorm.DB；jimu_nodb 构建里类型为 any 且恒为 nil。
	DB          DBHandle
	db          dbState
	Logger      *logger.Logger
	JobRegistry contract.JobRegistry
	Scheduler   *scheduler.CronScheduler
	EventBus    *event.EventBus
	HTTPClient  *httpclient.Client
	Reporter    reporter.Reporter
	DBCollector *metrics.DBCollector
	// otel 持有 OTel 追踪/指标/日志出口。类型与构造由两个同名 build-tag 文件提供：
	// 启用 OTel 的构建引用 OTel SDK 与导出器，0 能力构建为空实现。
	otel otelState
	// redis 持有 Redis 客户端与分布式锁。类型与构造由两个同名的 build-tag 文件提供：
	// 启用 Redis 的构建引用 go-redis，0 能力构建为空实现。
	redis redisState
}

func (c *Container) Start(context.Context) error { return nil }

// CollectMetrics 采集 DB 连接池与运行时指标（Prometheus 口径）。
// 未连库时是 no-op。
func (c *Container) CollectMetrics() {
	if c.DBCollector != nil {
		c.DBCollector.Collect()
	}
	metrics.CollectRuntime()
}

// HasMetricsCollector 报告是否已连库因而需要采集 DB 指标。
func (c *Container) HasMetricsCollector() bool { return c.DBCollector != nil }

func (c *Container) Stop(ctx context.Context) error {
	var result error
	result = errors.Join(result, c.redis.close())
	result = errors.Join(result, c.db.close())
	result = errors.Join(result, c.otel.shutdown(ctx))
	if c.Reporter != nil {
		// 优雅停机：给在途错误上报一个发送窗口
		c.Reporter.Flush(5 * time.Second)
	}
	if c.Logger != nil {
		result = errors.Join(result, c.Logger.Sync())
	}
	return result
}

func NewContainer(cfg *config.Config, sections config.SectionDecoder, capCfgs *CapabilityConfigs, caps []contract.Descriptor, enabled map[string]bool, drivers map[string][]string) (*Container, error) {
	// OpenObserve 日志通道：otel 启用时附加到 zap（初始化失败仅告警，不阻断启动）
	otel := newOTELState(cfg)
	log := logger.New(cfg.Log, otel.extraCores()...)

	requiredServices := requiredRuntimeServices(caps, drivers)
	database, err := newDBState(cfg, log, requiredServices)
	if err != nil {
		return nil, err
	}
	redis, err := newRedisState(cfg, log, requiredServices[contract.RuntimeServiceRedis])
	if err != nil {
		_ = database.close()
		return nil, err
	}

	// 调度器是内核件：其配置段由 queue 能力声明并已按启用集解码；未启用时取零值
	// （memory 存储），与下沉前的「不接线/默认行为」一致。
	var schedulerCfg scheduler.Config
	if c, ok := SectionOf[*scheduler.Config](capCfgs, scheduler.ConfigKey); ok {
		schedulerCfg = *c
	}
	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if schedulerCfg.Store == scheduler.StoreMySQL {
		schedStore, err = database.mysqlStore()
		if err != nil {
			_ = database.close()
			_ = redis.close()
			return nil, err
		}
	}
	var sched *scheduler.CronScheduler
	if schedulerCfg.Store == scheduler.StoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, redis.locker())
	} else {
		sched = scheduler.NewWithStore(log, schedStore, nil)
	}

	// 统一出站 HTTP client（oauth/webhook/breach 等外部调用复用）
	httpClient := httpclient.New(httpclient.Config{
		TimeoutSec:      cfg.HTTPClient.TimeoutSec,
		MaxRetries:      cfg.HTTPClient.MaxRetries,
		RetryIntervalMS: cfg.HTTPClient.RetryIntervalMS,
		RateLimitRate:   cfg.HTTPClient.RateLimitRate,
		RateLimitBurst:  cfg.HTTPClient.RateLimitBurst,
	})

	// Event Bus
	eventBus := event.New()

	// DB Metrics Collector
	dbCollector, err := database.collector()
	if err != nil {
		_ = database.close()
		_ = redis.close()
		return nil, err
	}

	// 错误上报：启用时输出结构化错误日志（日志链路接入 OpenObserve 后自动汇聚）
	// gRPC 服务端 panic 也经此上报（server recovery 拦截器）
	errorReporter := reporter.NewReporter(cfg.ErrorReport.Enabled, log.Errorw)

	return &Container{
		Config:            cfg,
		Sections:          sections,
		CapabilityConfigs: capCfgs,
		Enabled:           enabled,
		Capabilities:      caps,
		DB:                database.handle(),
		db:                database,
		redis:             redis,
		Logger:            log,
		Reporter:          errorReporter,
		JobRegistry:       sched,
		Scheduler:         sched,
		EventBus:          eventBus,
		DBCollector:       dbCollector,
		HTTPClient:        httpClient,
		otel:              otel,
	}, nil
}

func requiredRuntimeServices(caps []contract.Descriptor, drivers map[string][]string) map[contract.RuntimeService]bool {
	required := map[contract.RuntimeService]bool{}
	for _, capability := range caps {
		for _, service := range capability.RuntimeServices {
			required[service] = true
		}
		selected := drivers[capability.Name]
		for _, spec := range capability.DriverSpecs {
			if !slices.Contains(selected, spec.Name) {
				continue
			}
			for _, service := range spec.RuntimeServices {
				required[service] = true
			}
		}
	}
	return required
}

// HasRedis 报告容器是否持有 Redis 客户端（未编译 Redis 的构建恒为 false）。
func (c *Container) HasRedis() bool { return c.redis.hasClient() }
