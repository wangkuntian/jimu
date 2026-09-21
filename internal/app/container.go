package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	apikey "jimu/internal/capabilities/apikey"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/feature"
	grpcpkg "jimu/internal/capabilities/grpc"
	"jimu/internal/capabilities/notification"
	"jimu/internal/capabilities/outbox"
	"jimu/internal/capabilities/queue"
	queueinfra "jimu/internal/capabilities/queue/infrastructure"
	"jimu/internal/capabilities/storage"
	"jimu/internal/capabilities/uploadsec"
	userpkg "jimu/internal/capabilities/user"
	userinfrastructure "jimu/internal/capabilities/user/infrastructure"
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

type Container struct {
	Config *config.Config
	// Sections 按 YAML 点分键解码能力配置段（能力配置由能力自身声明，设计 §8）
	Sections config.SectionDecoder
	// Enabled 已启用能力名集合（含依赖闭包）
	Enabled map[string]bool
	// OutboxPublisher outbox 的发布器类型；outbox 能力未启用时为空（不接线）
	OutboxPublisher string
	DB              *gorm.DB
	Redis           redistore.Client
	Logger          *logger.Logger
	TracerProvider  *sdktrace.TracerProvider
	JobRegistry     contract.JobRegistry
	Scheduler       *scheduler.CronScheduler
	Lock            *redistore.Lock
	Storage         storage.Storage
	UploadScanner   uploadsec.Scanner
	Notification    notification.Dispatcher
	FeatureFlag     *feature.Manager
	WebSocketHub    *notification.Hub
	EventBus        *event.EventBus
	Outbox          *outbox.Outbox
	DBCollector     *observability.DBCollector
	HTTPClient      *httpclient.Client
	Cipher          *encryption.Cipher
	WorkerPool      *queue.WorkerPool
	APIKeyVerifier  *apikey.APIKeyVerifier
	// 泄露口令检查（HIBP）；auth.breach_check_enabled 关闭时为 nil
	BreachChecker contract.BreachChecker
	GRPCServer    *grpcpkg.Server
	Reporter      reporter.Reporter
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

func NewContainer(cfg *config.Config, sections config.SectionDecoder, enabled map[string]bool) (*Container, error) {
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
	var pendingWorkerPool *queue.WorkerPool

	// 雪花 ID：初始化全局生成器后再连库（hook 在 open 时注册）
	if err := db.InitSnowflake(cfg.ID.WorkerID); err != nil {
		return nil, err
	}
	dbConn, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return nil, err
	}
	// 字段级加密：注册全局 gorm hook（加密 email/phone 写入 + 盲索引 + 读取解密）
	cipher := encryption.New(cfg.Security.EncryptionKey)
	encryption.RegisterHooks(dbConn, cipher)
	rdb, err := redistore.ConnectWithRetry(cfg.Redis, log)
	if err != nil {
		return nil, err
	}

	lock := redistore.NewLock(rdb, "lock")

	// 能力配置段：由各能力声明默认值与校验（设计 §8）。
	// 调度器由 queue 能力用于作业调度，其配置段随之归 queue；两者都无条件加载
	// （container 的调度器实例与 outbox 接线不依赖能力启用集）。
	schedulerCfg, err := queue.LoadScheduler(sections)
	if err != nil {
		return nil, fmt.Errorf("init scheduler config: %w", err)
	}
	outboxCfg, err := outbox.Load(sections)
	if err != nil {
		return nil, fmt.Errorf("init outbox config: %w", err)
	}
	var queueCfg queue.Config
	if enabled["queue"] {
		loaded, err := queue.Load(sections)
		if err != nil {
			return nil, fmt.Errorf("init queue config: %w", err)
		}
		queueCfg = *loaded
	}
	// 跨能力校验（原 config.validateCommon 的 outbox.publisher=mq 依赖 queue.type）：
	// 两个能力都启用时才能在此判定。
	if enabled["outbox"] && enabled["queue"] && outboxCfg.UsesMQ() && !queue.SupportsOutboxMQ(queueCfg.Type) {
		return nil, fmt.Errorf("invalid queue.type %q for outbox.publisher %q", queueCfg.Type, outboxCfg.Publisher)
	}

	// outbox 接线名：能力未启用时为空（bootstrap 不接线）
	outboxWire := ""
	if enabled["outbox"] {
		outboxWire = outboxCfg.Publisher
	}

	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if schedulerCfg.Store == queue.SchedulerStoreMySQL {
		schedStore = scheduler.NewMySQLStore(dbConn)
	}
	var sched *scheduler.CronScheduler
	if schedulerCfg.Store == queue.SchedulerStoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, lock)
	} else {
		sched = scheduler.NewWithStore(log, schedStore, nil)
	}
	storageCfg, err := storage.Load(sections)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	storageSvc, err := storage.New(*storageCfg)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	// 文件上传病毒扫描器：上传能力未启用、或未开启扫描时为 nil（上传不扫描，向后兼容）。
	// 配置段仅在能力启用时解码与校验（未启用的能力配置段既不出现也不校验，设计 §8）。
	var uploadScanner uploadsec.Scanner
	if enabled["uploadsec"] {
		uploadCfg, err := uploadsec.Load(sections)
		if err != nil {
			return nil, fmt.Errorf("init upload config: %w", err)
		}
		uploadScanner = uploadCfg.Scanner()
	}

	// 统一出站 HTTP client（oauth/webhook 等外部调用复用）
	httpClient := httpclient.New(httpclient.Config{
		TimeoutSec:      cfg.HTTPClient.TimeoutSec,
		MaxRetries:      cfg.HTTPClient.MaxRetries,
		RetryIntervalMS: cfg.HTTPClient.RetryIntervalMS,
		RateLimitRate:   cfg.HTTPClient.RateLimitRate,
		RateLimitBurst:  cfg.HTTPClient.RateLimitBurst,
	})

	// 泄露口令检查（HIBP k-匿名范围查询）：默认关闭，启用时复用统一出站 client（超时/重试/熔断）
	var breachChecker contract.BreachChecker
	if cfg.Auth.BreachCheckEnabled {
		breachChecker = breach.New(httpClient)
	}

	notifier := notification.NewDispatcher()
	// WebSocket Hub（通知渠道 + 实时通信共用）
	wsHub := notification.NewHub()

	// 未配置真实发送渠道时，注册日志型兜底渠道，保证通知链路不报错且可观察
	var emailChannel notification.Notification = notification.NewLogChannel(notification.ChannelEmail, log)
	if cfg.Email.Enabled {
		emailChannel = notification.NewEmail(notification.EmailConfig{
			Host:     cfg.Email.Host,
			Port:     cfg.Email.Port,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
		})
	}
	notifier.Register(notification.ChannelEmail, emailChannel)

	// 短信：未配置真实发送时注册日志型兜底渠道，保证通知链路不报错且可观察
	var smsChannel notification.Notification = notification.NewLogChannel(notification.ChannelSMS, log)
	if cfg.SMS.Enabled {
		smsChannel = notification.NewSMS(notification.SMSConfig{
			Provider:  cfg.SMS.Provider,
			APIKey:    cfg.SMS.APIKey,
			APISecret: cfg.SMS.APISecret,
			SignName:  cfg.SMS.SignName,
		})
	}
	notifier.Register(notification.ChannelSMS, smsChannel)

	notifier.Register(notification.ChannelWebSocket, notification.NewWebSocket(wsHub))
	notifier.Register(notification.ChannelWebhook, notification.NewWebhook(notification.WebhookConfig{
		Headers:    map[string]string{},
		SignSecret: cfg.Notification.Webhook.SignSecret,
	}, httpClient))

	// Feature Flag
	featureMgr := feature.NewManager()
	// 注册默认特性开关
	featureMgr.Register(feature.Flag{
		Name:       "new_dashboard",
		Enabled:    false,
		Percentage: 0,
	})
	featureMgr.Register(feature.Flag{
		Name:       "beta_features",
		Enabled:    true,
		Percentage: 10, // 10% 灰度
	})

	// Event Bus
	eventBus := event.New()

	// Outbox
	outboxStore := outbox.NewMySQLStore(dbConn)
	var outboxPublisher outbox.Publisher
	switch outboxCfg.Publisher {
	case outbox.PublisherMQ:
		queueCfg.Redis = rdb
		q, err := queue.New(queueCfg)
		if err != nil {
			return nil, fmt.Errorf("init outbox queue: %w", err)
		}
		outboxPublisher = outbox.NewMQPublisher(q)
		consumer, ok := q.(queue.Consumer)
		if !ok {
			return nil, fmt.Errorf("queue %s does not implement consumer", queueCfg.Type)
		}
		store := queue.NewMySQLStore(
			queueinfra.NewMysqlJobRepository(dbConn),
			queueinfra.NewMysqlJobHistoryRepository(dbConn),
			queueinfra.NewMysqlDeadLetterRepository(dbConn),
		)
		workerPool := queue.NewWorkerPool(queue.DefaultWorkerConfig, consumer, store)
		// 延迟到 Container 构造后赋值（见 Step 3）
		pendingWorkerPool = workerPool
	default:
		outboxPublisher = outbox.NewEventBusPublisher(eventBus)
	}
	outboxProcessor := outbox.New(outboxStore, outboxPublisher)

	// DB Metrics Collector
	var dbCollector *observability.DBCollector
	if sqlDB, err := dbConn.DB(); err == nil {
		dbCollector = observability.NewDBCollector(sqlDB, "primary")
	}

	// API Key 验证器（服务/机器间认证，复用 admin api_keys 表）
	// 路由组按需挂载 apikey.APIKeyAuthMiddleware(c.APIKeyVerifier)
	apiKeyVerifier := apikey.NewAPIKeyVerifier(apikey.NewDBAPIKeyStore(dbConn))

	// 错误上报：启用时输出结构化错误日志（日志链路接入 OpenObserve 后自动汇聚）
	// gRPC 服务端 panic 也经此上报（server recovery 拦截器）
	errorReporter := reporter.NewReporter(cfg.ErrorReport, log.Errorw)

	// gRPC server（与 HTTP 双栈；bootstrap 在 grpc.enabled 时纳入生命周期）
	grpcServer, err := grpcpkg.New(grpcpkg.Config{
		Enabled:    cfg.GRPC.Enabled,
		Host:       cfg.GRPC.Host,
		Port:       cfg.GRPC.Port,
		TimeoutSec: cfg.GRPC.TimeoutSec,
		TLS:        cfg.GRPC.TLS,
	}, log, errorReporter)
	if err != nil {
		return nil, fmt.Errorf("init grpc server: %w", err)
	}
	// 业务示例：注册 UserInfoService，用户数据经 contract.UserinfoSource 端口读取
	// （user 能力提供适配实现，grpc 能力不直接依赖 user/domain）
	grpcServer.RegisterUserInfoService(userpkg.NewUserinfoSource(userinfrastructure.NewMysqlRepository(dbConn)))

	return &Container{
		Config:          cfg,
		Sections:        sections,
		Enabled:         enabled,
		OutboxPublisher: outboxWire,
		DB:              dbConn,
		Redis:           rdb,
		Logger:          log,
		Reporter:        errorReporter,
		JobRegistry:     sched,
		Scheduler:       sched,
		Lock:            lock,
		Storage:         storageSvc,
		UploadScanner:   uploadScanner,
		Notification:    notifier,
		FeatureFlag:     featureMgr,
		WebSocketHub:    wsHub,
		EventBus:        eventBus,
		Outbox:          outboxProcessor,
		DBCollector:     dbCollector,
		HTTPClient:      httpClient,
		Cipher:          cipher,
		WorkerPool:      pendingWorkerPool,
		APIKeyVerifier:  apiKeyVerifier,
		BreachChecker:   breachChecker,
		GRPCServer:      grpcServer,
		LogExporter:     logExporter,
	}, nil
}
