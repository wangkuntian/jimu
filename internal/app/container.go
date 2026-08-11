package app

import (
	"context"
	"errors"
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
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

	dbConn, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return nil, err
	}
	rdb, err := redistore.ConnectWithRetry(cfg.Redis, log)
	if err != nil {
		return nil, err
	}

	sched := scheduler.New(log)
	httpClient := httpplatform.NewClient(log)
	lock := redistore.NewLock(rdb, "lock")

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
	default:
		outboxPublisher = outbox.NewEventBusPublisher(eventBus)
	}
	outboxProcessor := outbox.New(outboxStore, outboxPublisher)

	// DB Metrics Collector
	var dbCollector *observability.DBCollector
	if sqlDB, err := dbConn.DB(); err == nil {
		dbCollector = observability.NewDBCollector(sqlDB, "primary")
	}

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
	}, nil
}
