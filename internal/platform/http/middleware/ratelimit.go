package middleware

import (
	"sync"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter 基于令牌桶的限流器
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter 创建限流器
// r: 每秒生成的令牌数, b: 桶的最大容量
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// getLimiter 获取或创建 IP 对应的限流器
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}
	return limiter
}

// Limit 限流中间件
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			response.Fail(c, errors.New(errors.CodeRateLimited, "too many requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// GlobalRateLimit 全局限流（可配置速率和桶容量）
func GlobalRateLimit(rps, burst int) gin.HandlerFunc {
	if rps <= 0 {
		rps = 100
	}
	if burst <= 0 {
		burst = 200
	}
	limiter := NewRateLimiter(rate.Limit(rps), burst)

	return limiter.Limit()
}

// StrictRateLimit 严格限流（每秒 10 次，突发 20）
func StrictRateLimit() gin.HandlerFunc {
	limiter := NewRateLimiter(10, 20)
	return limiter.Limit()
}
