package middleware

import (
	"fmt"
	"time"

	"jimu/internal/kernel/auth"
	httpmw "jimu/internal/kernel/http/middleware"
	redistore "jimu/internal/kernel/redis"

	"github.com/gin-gonic/gin"
)

// APIKeyRateLimitMiddleware API Key 维度限流（Redis 滑动窗口）。
// 需前置 APIKeyAuthMiddleware（限流 key 用 Key ID，不用明文凭证）；
// 未携带 API Key 的请求跳过。
func APIKeyRateLimitMiddleware(client redistore.Client, limit int, window time.Duration) gin.HandlerFunc {
	return httpmw.NewUserRateLimiter(client, limit, window,
		httpmw.WithKeyPrefix("ratelimit:apikey"),
		httpmw.WithKeyFunc(func(c *gin.Context) string {
			if apiKey, ok := auth.APIKeyFromContext(c.Request.Context()); ok && apiKey != nil {
				return fmt.Sprintf("apikey:%d", apiKey.ID)
			}
			return ""
		}),
	).Middleware()
}
