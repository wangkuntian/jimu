package app

import (
	"context"
	"errors"

	"jimu/internal/config"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
	redistore "jimu/internal/platform/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *logger.Logger
}

func (c *Container) Start(context.Context) error { return nil }

func (c *Container) Stop(context.Context) error {
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
	if c.Logger != nil {
		result = errors.Join(result, c.Logger.Sync())
	}
	return result
}

func NewContainer(cfg *config.Config) (*Container, error) {
	dbConn, err := db.New(cfg.DB)
	if err != nil {
		return nil, err
	}
	rdb := redistore.New(cfg.Redis)
	log := logger.New(cfg.Log)

	return &Container{
		Config: cfg,
		DB:     dbConn,
		Redis:  rdb,
		Logger: log,
	}, nil
}
