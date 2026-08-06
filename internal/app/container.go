package app

import (
	"context"
	"errors"
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/observability"
	"jimu/internal/platform/scheduler"
	"jimu/internal/platform/storage"
	httpplatform "jimu/internal/platform/http"
	redistore "jimu/internal/platform/redis"

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
	}, nil
}
