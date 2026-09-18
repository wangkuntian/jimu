package middleware

import (
	"time"

	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequestsShedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "jimu",
	Subsystem: "http",
	Name:      "requests_shed_total",
	Help:      "Total number of requests rejected by concurrency limit",
})

// ConcurrencyLimit 并发处理上限（负载保护）。
// max <= 0 表示不限制；waitTimeout <= 0 表示不排队、立即拒绝（快速失败）。
// 超过上限返回 1010/503，客户端应稍后重试；等待期间请求上下文结束则直接放弃。
func ConcurrencyLimit(max int, waitTimeout time.Duration) gin.HandlerFunc {
	if max <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	sem := make(chan struct{}, max)
	release := func() { <-sem }

	return func(c *gin.Context) {
		if waitTimeout <= 0 {
			select {
			case sem <- struct{}{}:
				defer release()
				c.Next()
			default:
				shed(c)
			}
			return
		}

		timer := time.NewTimer(waitTimeout)
		defer timer.Stop()
		select {
		case sem <- struct{}{}:
			defer release()
			c.Next()
		case <-c.Request.Context().Done():
			// 客户端断开或上游已超时，无需再写响应
			c.Abort()
		case <-timer.C:
			shed(c)
		}
	}
}

// shed 拒绝请求并计数
func shed(c *gin.Context) {
	httpRequestsShedTotal.Inc()
	response.Fail(c, apperrors.New(apperrors.CodeServiceUnavailable, "server busy"))
	c.Abort()
}
