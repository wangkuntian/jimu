package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "jimu",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	httpRequestsInflight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "http",
		Name:      "requests_inflight",
		Help:      "Current number of in-flight HTTP requests",
	})
)

// Metrics HTTP 指标中间件。
// 标签使用路由模板（c.FullPath）控制基数，如 /api/v1/users/:id；
// 未匹配路由（404、被外层 Recovery 吞掉的 panic）回退到原始路径。
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method

		httpRequestsInflight.Inc()
		defer httpRequestsInflight.Dec()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		labels := prometheus.Labels{"method": method, "path": path, "status": status}
		httpRequestsTotal.With(labels).Inc()
		httpRequestDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}
