package redis

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/redis/go-redis/v9"
)

// New 创建 Redis 客户端（仅建立连接，不验证）
func New(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSec) * time.Second,
		MaxRetries:   3,
	})
}

// ConnectWithRetry 带重试的 Redis 连接（验证连通性）
func ConnectWithRetry(cfg config.RedisConfig, log *logger.Logger) (*redis.Client, error) {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	interval := cfg.RetryIntervalSec
	if interval <= 0 {
		interval = 3
	}

	client := New(cfg)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			if log != nil {
				log.Info("redis connected", "attempt", attempt)
			}
			return client, nil
		}

		if log != nil {
			log.Warn("retrying redis connection",
				"attempt", attempt,
				"max_retries", maxRetries,
				"interval_sec", interval,
				"error", err.Error(),
			)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}

	return nil, fmt.Errorf("redis connection failed after %d attempts", maxRetries)
}
