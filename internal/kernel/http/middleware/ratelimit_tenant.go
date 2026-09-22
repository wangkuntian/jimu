package middleware

import (
	"fmt"
	"time"

	redistore "jimu/internal/kernel/redis"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
)

// TenantRateLimitMiddleware 租户维度限流（Redis 滑动窗口）。
// 租户来自请求上下文（JWT tid / API Key 归属），平台级视角（tid=0）跳过；
// Redis 异常时 fail-open 放行。
//
// 组合根在装配受保护中间件链时使用；类型属内核（不依赖任何能力包），
// 实现沿用 kernel/http/middleware.NewUserRateLimiter。
func TenantRateLimitMiddleware(client redistore.Client, limit int, window time.Duration) gin.HandlerFunc {
	return NewUserRateLimiter(client, limit, window,
		WithKeyPrefix("ratelimit:tenant"),
		WithKeyFunc(func(c *gin.Context) string {
			if tid := tenant.FromContext(c.Request.Context()); tid != 0 {
				return fmt.Sprintf("tenant:%d", tid)
			}
			return ""
		}),
	).Middleware()
}
