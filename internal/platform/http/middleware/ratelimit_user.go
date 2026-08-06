package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// UserRateLimiter 基于用户/API Key 的限流器（Redis 滑动窗口）
type UserRateLimiter struct {
	client  *redis.Client
	prefix  string
	limit   int                       // 窗口内允许的最大请求数
	window  time.Duration             // 窗口大小
	keyFunc func(*gin.Context) string // 提取限流 key 的函数
}

// UserRateLimiterOption 配置选项
type UserRateLimiterOption func(*UserRateLimiter)

// WithKeyPrefix 设置 Redis key 前缀
func WithKeyPrefix(prefix string) UserRateLimiterOption {
	return func(r *UserRateLimiter) {
		r.prefix = prefix
	}
}

// WithKeyFunc 设置自定义 key 提取函数
func WithKeyFunc(fn func(*gin.Context) string) UserRateLimiterOption {
	return func(r *UserRateLimiter) {
		r.keyFunc = fn
	}
}

// NewUserRateLimiter 创建用户维度限流器
func NewUserRateLimiter(client *redis.Client, limit int, window time.Duration, opts ...UserRateLimiterOption) *UserRateLimiter {
	r := &UserRateLimiter{
		client:  client,
		prefix:  "ratelimit:user",
		limit:   limit,
		window:  window,
		keyFunc: defaultKeyFunc,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// defaultKeyFunc 默认 key 提取：优先 user_id，其次 API Key，最后 IP
func defaultKeyFunc(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	if apiKey := c.GetHeader("X-Api-Key"); apiKey != "" {
		return fmt.Sprintf("apikey:%s", apiKey)
	}
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// Middleware 返回 Gin 中间件
func (r *UserRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := r.prefix + ":" + r.keyFunc(c)
		allowed, remaining, resetAt, err := r.allow(c.Request.Context(), key)
		if err != nil {
			// Redis 错误时放行（fail-open），避免服务不可用
			c.Next()
			return
		}

		// 设置限流响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(r.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(time.Until(resetAt).Seconds())+1))
			response.Fail(c, errors.New(errors.CodeRateLimited, "too many requests"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow 使用 Redis 滑动窗口算法判断是否允许请求
// 返回：是否允许、剩余次数、重置时间、错误
func (r *UserRateLimiter) allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Add(-r.window)

	pipe := r.client.Pipeline()

	// 移除窗口外的记录
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	// 统计窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)
	// 添加当前请求
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})
	// 设置 key 过期时间
	pipe.Expire(ctx, key, r.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, now, err
	}

	count := int(countCmd.Val())
	remaining := r.limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	resetAt := now.Add(r.window)
	allowed := count < r.limit

	return allowed, remaining, resetAt, nil
}

// UserRateLimitMiddleware 便捷函数：创建用户维度限流中间件
// limit: 窗口内最大请求数
// window: 窗口大小
func UserRateLimitMiddleware(client *redis.Client, limit int, window time.Duration, opts ...UserRateLimiterOption) gin.HandlerFunc {
	limiter := NewUserRateLimiter(client, limit, window, opts...)
	return limiter.Middleware()
}
