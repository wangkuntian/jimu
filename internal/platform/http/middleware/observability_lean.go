//go:build jimu_lean

package middleware

import (
	"time"

	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Metrics 在 lean 构建中不采集 Prometheus 指标。
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// ConcurrencyLimit 在 lean 构建中不加载并发保护指标；上限本身仍生效，
// 只是被拒绝的请求不计数。
func ConcurrencyLimit(max int, waitTimeout time.Duration) gin.HandlerFunc {
	if max <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	sem := make(chan struct{}, max)
	return func(c *gin.Context) {
		if waitTimeout <= 0 {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				c.Next()
			default:
				response.Fail(c, apperrors.New(apperrors.CodeServiceUnavailable, "server busy"))
				c.Abort()
			}
			return
		}
		timer := time.NewTimer(waitTimeout)
		defer timer.Stop()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		case <-c.Request.Context().Done():
			c.Abort()
		case <-timer.C:
			response.Fail(c, apperrors.New(apperrors.CodeServiceUnavailable, "server busy"))
			c.Abort()
		}
	}
}
