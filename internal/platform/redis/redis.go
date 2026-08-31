package redis

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/redis/go-redis/v9"
)

// Client 是所有 Redis 部署模式（单机/哨兵/集群）客户端的统一接口。
// go-redis 的 *redis.Client、*redis.ClusterClient 均实现本接口，
// 便于上层组件与具体部署模式解耦。
type Client interface {
	redis.UniversalClient
	Pipeline() redis.Pipeliner
	TxPipeline() redis.Pipeliner
}

// New 创建 Redis 客户端（仅建立连接，不验证）。
// 根据 cfg.Mode 选择构造方式：
//   - single：单机 redis.NewClient
//   - sentinel：哨兵高可用 redis.NewFailoverClient
//   - cluster：集群 redis.NewClusterClient
//
// 统一返回 Client 接口（三种客户端均实现）。
func New(cfg config.RedisConfig) Client {
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}
	minIdle := cfg.MinIdleConns
	if minIdle <= 0 {
		minIdle = 0
	}
	timeout := func(sec int) time.Duration {
		if sec <= 0 {
			return 3 * time.Second
		}
		return time.Duration(sec) * time.Second
	}

	switch cfg.Mode {
	case config.RedisModeSentinel:
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       cfg.MasterName,
			SentinelAddrs:    cfg.SentinelAddrs,
			SentinelPassword: cfg.SentinelPassword,
			Password:         cfg.Password,
			DB:               cfg.DB,
			PoolSize:         poolSize,
			MinIdleConns:     minIdle,
			ReadTimeout:      timeout(cfg.ReadTimeoutSec),
			WriteTimeout:     timeout(cfg.WriteTimeoutSec),
			MaxRetries:       3,
		})
	case config.RedisModeCluster:
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddrs,
			Password:     cfg.Password,
			PoolSize:     poolSize,
			MinIdleConns: minIdle,
			ReadTimeout:  timeout(cfg.ReadTimeoutSec),
			WriteTimeout: timeout(cfg.WriteTimeoutSec),
			MaxRetries:   3,
		})
	default:
		return redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     poolSize,
			MinIdleConns: minIdle,
			ReadTimeout:  timeout(cfg.ReadTimeoutSec),
			WriteTimeout: timeout(cfg.WriteTimeoutSec),
			MaxRetries:   3,
		})
	}
}

// ConnectWithRetry 带重试的 Redis 连接（验证连通性）
func ConnectWithRetry(cfg config.RedisConfig, log *logger.Logger) (Client, error) {
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
				log.Info("redis connected", "attempt", attempt, "mode", cfg.Mode)
			}
			return client, nil
		}

		if log != nil {
			log.Warn("retrying redis connection",
				"attempt", attempt,
				"max_retries", maxRetries,
				"interval_sec", interval,
				"mode", cfg.Mode,
				"error", err.Error(),
			)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}

	return nil, fmt.Errorf("redis connection failed after %d attempts", maxRetries)
}
