//go:build jimu_noredis

package app

import (
	"jimu/internal/config"
	"jimu/internal/kernel/lock"
	"jimu/internal/kernel/logger"
)

// redisState 是未启用 Redis 的构建的空实现：不引用 go-redis。
type redisState struct{}

func newRedisState(*config.Config, *logger.Logger, bool) (redisState, error) {
	return redisState{}, nil
}

func (redisState) locker() lock.Locker { return nil }

func (redisState) close() error { return nil }

// HasRedis 报告当前构建是否连上了 Redis。
func (redisState) hasClient() bool { return false }
