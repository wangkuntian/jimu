package app

import (
	"context"
	"errors"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/observability"
	"jimu/internal/platform/scheduler"
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

	return &Container{
		Config:      cfg,
		DB:          dbConn,
		Redis:       rdb,
		Logger:      log,
		JobRegistry: sched,
		Scheduler:   sched,
		HTTPClient:  httpClient,
	}, nil
}
