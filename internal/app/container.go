package app

import (
	"context"
	"errors"
	"log"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/db"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/observability"
	redistore "jimu/internal/kernel/redis"
	"jimu/internal/kernel/reporter"
	"jimu/internal/kernel/scheduler"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
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
	Capabilities   []contract.Descriptor
	DB             *gorm.DB
	Redis          redistore.Client
	Logger         *logger.Logger
	TracerProvider *sdktrace.TracerProvider
	JobRegistry    contract.JobRegistry
	Scheduler      *scheduler.CronScheduler
	Lock           *redistore.Lock
	EventBus       *event.EventBus
	DBCollector    *observability.DBCollector
	HTTPClient     *httpclient.Client
	Reporter       reporter.Reporter
	// 观测出口（OTLP → OpenObserve；未启用时为 nil）
	MetricsPusher *observability.MetricsPusher
	LogExporter   *observability.LogExporter
}

func (c *Container) Start(context.Context) error { return nil }

func (c *Container) Stop(ctx context.Context) error {
	var result error
	if c.Redis != nil {
		result = errors.Join(result, c.Redis.Close())
	}
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			result = errors.Join(result, err)
		} else {
			result = errors.Join(result, sqlDB.Close())
		}
	}
	if c.TracerProvider != nil {
		result = errors.Join(result, observability.ShutdownTracing(ctx, c.TracerProvider))
	}
	if c.MetricsPusher != nil {
		result = errors.Join(result, c.MetricsPusher.Shutdown(ctx))
	}
	if c.LogExporter != nil {
		result = errors.Join(result, c.LogExporter.Shutdown(ctx))
	}
	if c.Reporter != nil {
		// 优雅停机：给在途错误上报一个发送窗口
		c.Reporter.Flush(5 * time.Second)
	}
	if c.Logger != nil {
		result = errors.Join(result, c.Logger.Sync())
	}
	return result
}

func NewContainer(cfg *config.Config, sections config.SectionDecoder, capCfgs *CapabilityConfigs, caps []contract.Descriptor, enabled map[string]bool) (*Container, error) {
	// OpenObserve 日志通道：otel 启用时附加到 zap（初始化失败仅告警，不阻断启动）
	var (
		logExporter *observability.LogExporter
		extraCores  []zapcore.Core
	)
	if cfg.OTEL.Enabled && cfg.OTEL.LogsEnabled {
		var err error
		logExporter, err = observability.NewLogExporter(context.Background(), cfg.OTEL)
		if err != nil {
			log.Printf("openobserve logs exporter init failed: %v", err)
		} else {
			extraCores = append(extraCores, logExporter.ZapCore(zapcore.DebugLevel))
		}
	}
	log := logger.New(cfg.Log, extraCores...)

	// 雪花 ID：初始化全局生成器后再连库（hook 在 open 时注册）
	if err := db.InitSnowflake(cfg.ID.WorkerID); err != nil {
		return nil, err
	}
	dbConn, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return nil, err
	}
	rdb, err := redistore.ConnectWithRetry(cfg.Redis, log)
	if err != nil {
		return nil, err
	}

	lock := redistore.NewLock(rdb, "lock")

	// 调度器是内核件：其配置段由 queue 能力声明并已按启用集解码；未启用时取零值
	// （memory 存储），与下沉前的「不接线/默认行为」一致。
	var schedulerCfg scheduler.Config
	if c, ok := SectionOf[*scheduler.Config](capCfgs, scheduler.ConfigKey); ok {
		schedulerCfg = *c
	}
	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if schedulerCfg.Store == scheduler.StoreMySQL {
		schedStore = scheduler.NewMySQLStore(dbConn)
	}
	var sched *scheduler.CronScheduler
	if schedulerCfg.Store == scheduler.StoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, lock)
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
	var dbCollector *observability.DBCollector
	if sqlDB, err := dbConn.DB(); err == nil {
		dbCollector = observability.NewDBCollector(sqlDB, "primary")
	}

	// 错误上报：启用时输出结构化错误日志（日志链路接入 OpenObserve 后自动汇聚）
	// gRPC 服务端 panic 也经此上报（server recovery 拦截器）
	errorReporter := reporter.NewReporter(cfg.ErrorReport, log.Errorw)

	return &Container{
		Config:            cfg,
		Sections:          sections,
		CapabilityConfigs: capCfgs,
		Enabled:           enabled,
		Capabilities:      caps,
		DB:                dbConn,
		Redis:             rdb,
		Logger:            log,
		Reporter:          errorReporter,
		JobRegistry:       sched,
		Scheduler:         sched,
		Lock:              lock,
		EventBus:          eventBus,
		DBCollector:       dbCollector,
		HTTPClient:        httpClient,
		LogExporter:       logExporter,
	}, nil
}
