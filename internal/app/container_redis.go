//go:build !jimu_noredis

package app

import (
	"jimu/internal/config"
	"jimu/internal/kernel/lock"
	"jimu/internal/kernel/logger"
	redistore "jimu/internal/platform/redisclient"
)

// redisState 持有 Redis 客户端与分布式锁。
// 只在 jimu_redis 构建里引用 go-redis；0 能力项目用 container_redis_empty.go。
type redisState struct {
	client redistore.Client
	lock   lock.Locker
}

func newRedisState(cfg *config.Config, log *logger.Logger, required bool) (redisState, error) {
	if !required {
		return redisState{}, nil
	}
	client, err := redistore.ConnectWithRetry(cfg.Redis, log)
	if err != nil {
		return redisState{}, err
	}
	return redisState{client: client, lock: redistore.NewLock(client, "lock")}, nil
}

func (s redisState) locker() lock.Locker {
	if s.lock == nil {
		return nil
	}
	return s.lock
}

func (s redisState) close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

// RedisClient 返回带 go-redis 方法集的客户端。
func (c *Container) RedisClient() redistore.Client { return c.redis.client }

// RedisLock 返回分布式锁。
func (c *Container) RedisLock() *redistore.Lock {
	lock, _ := c.redis.lock.(*redistore.Lock)
	return lock
}

func (s redisState) hasClient() bool { return s.client != nil }

// WithRedisForTest 注入 Redis 客户端（仅测试使用）。
func (c *Container) WithRedisForTest(client redistore.Client) *Container {
	c.redis = redisState{client: client, lock: redistore.NewLock(client, "lock")}
	return c
}
