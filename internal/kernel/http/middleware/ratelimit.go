package middleware

import (
	"fmt"
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

// cleanupThreshold 触发惰性清理的 visitor 数量上限（内存有界）
const cleanupThreshold = 10000

// getLimiter 获取或创建 IP 对应的限流器
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
		// 超阈值时清理满桶（长时间未消费）的条目，防 map 无限增长
		if len(rl.visitors) > cleanupThreshold {
			rl.cleanupLocked()
		}
	}
	return limiter
}

// cleanupLocked 删除满桶条目（满桶 = 未消费 = IP 已空闲）
func (rl *RateLimiter) cleanupLocked() {
	for ip, l := range rl.visitors {
		if l.Tokens() >= float64(rl.burst) {
			delete(rl.visitors, ip)
		}
	}
}

// Limit 限流中间件（带 RFC 6585 标准响应头）
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.rate)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", fmt.Sprintf("%d", int(60/rl.rate)+1))
			response.Fail(c, errors.New(errors.CodeRateLimited, "too many requests"))
			c.Abort()
			return
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.rate)))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int(limiter.Tokens())))
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
