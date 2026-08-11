package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	admininfra "jimu/internal/modules/admin/infrastructure"
	"jimu/internal/platform/captcha"
	"jimu/internal/platform/db"
	"jimu/internal/platform/event"
	"jimu/internal/platform/feature"
	httpplatform "jimu/internal/platform/http"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/observability"
	"jimu/internal/platform/outbox"
	"jimu/internal/platform/queue"
	redistore "jimu/internal/platform/redis"
	"jimu/internal/platform/scheduler"
	"jimu/internal/platform/storage"

	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"
)

type Container struct {
	Config         *config.Config
	DB             *gorm.DB
	Redis          *redis.Client
	Logger         *logger.Logger
	TracerProvider *sdktrace.TracerProvider
	JobRegistry    contract.JobRegistry
	Scheduler      *scheduler.CronScheduler
	HTTPClient     *httpplatform.Client
	Lock           *redistore.Lock
	Storage        storage.Storage
	Notification   notification.Dispatcher
	FeatureFlag    *feature.Manager
	WebSocketHub   *notification.Hub
	EventBus       *event.EventBus
	Outbox         *outbox.Outbox
	DBCollector    *observability.DBCollector
	Captcha        *captcha.Service
	WorkerPool     *queue.WorkerPool
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
	if c.Logger != nil {
		result = errors.Join(result, c.Logger.Sync())
	}
	return result
}

func NewContainer(cfg *config.Config) (*Container, error) {
	log := logger.New(cfg.Log)
	var pendingWorkerPool *queue.WorkerPool

	dbConn, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return nil, err
	}
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
	httpClient := httpplatform.NewClient(log)

	store, err := storage.New(storage.Config{
		Type:    storage.StorageType(cfg.Storage.Type),
		BaseDir: cfg.Storage.BaseDir,
		BaseURL: cfg.Storage.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	notifier := notification.NewDispatcher()

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

	// WebSocket Hub
	wsHub := notification.NewHub()

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
		if cfg.Queue.Type == string(queue.TypeKafka) || cfg.Queue.Type == string(queue.TypeRabbitMQ) {
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
		}
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

	return &Container{
		Config:       cfg,
		DB:           dbConn,
		Redis:        rdb,
		Logger:       log,
		JobRegistry:  sched,
		Scheduler:    sched,
		HTTPClient:   httpClient,
		Lock:         lock,
		Storage:      store,
		Notification: notifier,
		FeatureFlag:  featureMgr,
		WebSocketHub: wsHub,
		EventBus:     eventBus,
		Outbox:       outboxProcessor,
		DBCollector:  dbCollector,
		Captcha:      captchaSvc,
		WorkerPool:   pendingWorkerPool,
	}, nil
}
