package app

import (
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
