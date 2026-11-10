//go:build jimu_noredis

package app

import (
	"jimu/internal/config"
	"jimu/internal/platform/health"

	"github.com/gin-gonic/gin"
)

// redisMiddlewares 在未编译 Redis 的构建里恒为空。
func redisMiddlewares(*Container, *config.Config) []gin.HandlerFunc { return nil }

// redisHealthCheck 在未编译 Redis 的构建里恒为空。
func redisHealthCheck(*Container) []health.Checker { return nil }

// instrumentRedisTracing 在未编译 Redis 的构建里是 no-op。
func instrumentRedisTracing(*Container) error { return nil }
