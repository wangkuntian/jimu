package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	admininfra "jimu/internal/modules/admin/infrastructure"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/captcha"
	"jimu/internal/platform/db"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/event"
	"jimu/internal/platform/feature"
	grpcpkg "jimu/internal/platform/grpc"
	platformhttp "jimu/internal/platform/http"
	"jimu/internal/platform/httpclient"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/observability"
	"jimu/internal/platform/outbox"
	"jimu/internal/platform/queue"
	redistore "jimu/internal/platform/redis"
	"jimu/internal/platform/reporter"
	"jimu/internal/platform/scheduler"
	"jimu/internal/platform/storage"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"
)

type Container struct {
	Config         *config.Config
	DB             *gorm.DB
	Redis          redistore.Client
	Logger         *logger.Logger
	TracerProvider *sdktrace.TracerProvider
	JobRegistry    contract.JobRegistry
	Scheduler      *scheduler.CronScheduler
	Lock           *redistore.Lock
	Storage        storage.Storage
	UploadScanner  platformhttp.Scanner
	Notification   notification.Dispatcher
	FeatureFlag    *feature.Manager
	WebSocketHub   *notification.Hub
	EventBus       *event.EventBus
	Outbox         *outbox.Outbox
	DBCollector    *observability.DBCollector
	HTTPClient     *httpclient.Client
	Captcha        *captcha.Service
	Cipher         *encryption.Cipher
	WorkerPool     *queue.WorkerPool
	APIKeyVerifier *auth.APIKeyVerifier
	GRPCServer     *grpcpkg.Server
	Reporter       reporter.Reporter
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
	if c.Reporter != nil {
		// 优雅停机：给在途错误上报一个发送窗口
		c.Reporter.Flush(5 * time.Second)
	}
	if c.Logger != nil {
		result = errors.Join(result, c.Logger.Sync())
	}
	return result
}

func NewContainer(cfg *config.Config) (*Container, error) {
	log := logger.New(cfg.Log)
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
	db.RegisterEncryptionHooks(dbConn, cipher)
	rdb, err := redistore.ConnectWithRetry(cfg.Redis, log)
	if err != nil {
		return nil, err
	}

	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if cfg.Scheduler.Store == config.SchedulerStoreMySQL {
		schedStore = scheduler.NewMySQLStore(dbConn)
	}
	lock := redistore.NewLock(rdb, "lock")
	var sched *scheduler.CronScheduler
	if cfg.Scheduler.Store == config.SchedulerStoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, lock)
	} else {
		sched = scheduler.NewWithStore(log, schedStore, nil)
	}
	storageSvc, err := storage.New(storage.Config{
		Type:      storage.StorageType(cfg.Storage.Type),
		BaseDir:   cfg.Storage.BaseDir,
		BaseURL:   cfg.Storage.BaseURL,
		Endpoint:  cfg.Storage.Endpoint,
		Region:    cfg.Storage.Region,
		Bucket:    cfg.Storage.Bucket,
		AccessKey: cfg.Storage.AccessKey,
		SecretKey: cfg.Storage.SecretKey,
		PathStyle: cfg.Storage.PathStyle,
	})
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	// 文件上传病毒扫描器：未启用时为 nil（上传不扫描，向后兼容）
	var uploadScanner platformhttp.Scanner
	if cfg.Upload.ClamAV.Enabled {
		uploadScanner = platformhttp.NewClamAVScanner(platformhttp.ClamAVConfig{
			Address: cfg.Upload.ClamAV.Address,
			Timeout: time.Duration(cfg.Upload.ClamAV.TimeoutSec) * time.Second,
		})
	}

	// 统一出站 HTTP client（oauth/webhook 等外部调用复用）
	httpClient := httpclient.New(httpclient.Config{
		TimeoutSec:      cfg.HTTPClient.TimeoutSec,
		MaxRetries:      cfg.HTTPClient.MaxRetries,
		RetryIntervalMS: cfg.HTTPClient.RetryIntervalMS,
		RateLimitRate:   cfg.HTTPClient.RateLimitRate,
		RateLimitBurst:  cfg.HTTPClient.RateLimitBurst,
	})

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
	switch cfg.Outbox.Publisher {
	case config.OutboxPublisherMQ:
		q, err := queue.New(queue.Config{
			Type:  queue.Type(cfg.Queue.Type),
			Redis: rdb,
			Kafka: queue.KafkaConfig{
				Brokers: cfg.Queue.Kafka.Brokers,
				Topic:   cfg.Queue.Kafka.Topic,
				GroupID: cfg.Queue.Kafka.GroupID,
			},
			RabbitMQ: queue.RabbitMQConfig{
				URL:       cfg.Queue.RabbitMQ.URL,
				QueueName: cfg.Queue.RabbitMQ.Queue,
				Exchange:  cfg.Queue.RabbitMQ.Exchange,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("init outbox queue: %w", err)
		}
		outboxPublisher = outbox.NewMQPublisher(q)
		consumer, ok := q.(queue.Consumer)
		if !ok {
			return nil, fmt.Errorf("queue %s does not implement consumer", cfg.Queue.Type)
		}
		store := queue.NewMySQLStore(
			admininfra.NewMysqlJobRepository(dbConn),
			admininfra.NewMysqlJobHistoryRepository(dbConn),
			admininfra.NewMysqlDeadLetterRepository(dbConn),
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

	// Captcha 验证码服务（平台能力，非业务模块；auth 模块消费）
	captchaSvc := captcha.NewService(rdb, time.Duration(cfg.Captcha.TTLMin)*time.Minute)

	// API Key 验证器（服务/机器间认证，复用 admin api_keys 表）
	// 路由组按需挂载 auth.APIKeyAuthMiddleware(c.APIKeyVerifier)
	apiKeyVerifier := auth.NewAPIKeyVerifier(auth.NewDBAPIKeyStore(dbConn))

	// gRPC server（与 HTTP 双栈；bootstrap 在 grpc.enabled 时纳入生命周期）
	grpcServer := grpcpkg.New(grpcpkg.Config{
		Enabled: cfg.GRPC.Enabled,
		Host:    cfg.GRPC.Host,
		Port:    cfg.GRPC.Port,
	}, log)
	// 业务示例：注册 UserInfoService（真实业务模块可在此注入自己的 service）
	grpcServer.RegisterUserInfoService(dbConn)

	// 错误追踪上报（Sentry 等）：未启用时为空实现，零开销。
	// Environment 优先取配置；未配置时回退应用元数据环境（APP_ENV）。
	reportCfg := cfg.ErrorReport
	if reportCfg.Environment == "" {
		reportCfg.Environment = cfg.Environment
	}
	errorReporter := reporter.NewReporter(reportCfg, log.Errorw)

	return &Container{
		Config:         cfg,
		DB:             dbConn,
		Redis:          rdb,
		Logger:         log,
		Reporter:       errorReporter,
		JobRegistry:    sched,
		Scheduler:      sched,
		Lock:           lock,
		Storage:        storageSvc,
		UploadScanner:  uploadScanner,
		Notification:   notifier,
		FeatureFlag:    featureMgr,
		WebSocketHub:   wsHub,
		EventBus:       eventBus,
		Outbox:         outboxProcessor,
		DBCollector:    dbCollector,
		HTTPClient:     httpClient,
		Captcha:        captchaSvc,
		Cipher:         cipher,
		WorkerPool:     pendingWorkerPool,
		APIKeyVerifier: apiKeyVerifier,
		GRPCServer:     grpcServer,
	}, nil
}
