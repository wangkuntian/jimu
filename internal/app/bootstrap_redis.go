//go:build !jimu_noredis

package app

import (
	"context"
	"time"

	"jimu/internal/config"
	"jimu/internal/kernel/health"
	middlewareidem "jimu/internal/kernel/http/middleware/idempotency"
	middlewarerate "jimu/internal/kernel/http/middleware/ratelimit"

	redisotel "github.com/redis/go-redis/extra/redisotel/v9"

	"github.com/gin-gonic/gin"
)

// redisMiddlewares 返回依赖 Redis 的受保护中间件（租户限流、幂等）。
func redisMiddlewares(container *Container, cfg *config.Config) []gin.HandlerFunc {
	if container.redis.client == nil {
		return nil
	}
	var extra []gin.HandlerFunc
	if cfg.RateLimit.Tenant.Enabled && cfg.RateLimit.Tenant.Limit > 0 {
		extra = append(extra, middlewarerate.TenantRateLimitMiddleware(
			container.redis.client,
			cfg.RateLimit.Tenant.Limit,
			time.Duration(cfg.RateLimit.Tenant.WindowSec)*time.Second,
		))
	}
	if cfg.Security.IdempotencyEnabled {
		extra = append(extra, middlewareidem.IdempotencyMiddleware(
			container.redis.client,
			time.Duration(cfg.Security.IdempotencyTTLSec)*time.Second,
		))
	}
	return extra
}

// redisHealthCheck 把 Redis 连通性并入 /readyz。
func redisHealthCheck(container *Container) []health.Checker {
	if container.redis.client == nil {
		return nil
	}
	return []health.Checker{health.CheckerFunc(func(ctx context.Context) error {
		return container.redis.client.Ping(ctx).Err()
	})}
}

// instrumentRedisTracing 给 Redis 客户端挂 OTel 追踪插件。
func instrumentRedisTracing(container *Container) error {
	if container.redis.client == nil {
		return nil
	}
	return redisotel.InstrumentTracing(container.redis.client)
}
