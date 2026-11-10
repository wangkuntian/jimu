package middleware

import (
	"fmt"
	"time"

	"jimu/internal/platform/http/ctxkeys"
	httpmw "jimu/internal/platform/http/middleware/ratelimit"
	redistore "jimu/internal/platform/redisclient"

	"github.com/gin-gonic/gin"
)

// APIKeyRateLimitMiddleware API Key 维度限流（Redis 滑动窗口）。
// 需前置 APIKeyAuthMiddleware（限流 key 用 Key ID，不用明文凭证）；
// 未携带 API Key 的请求跳过。
func APIKeyRateLimitMiddleware(client redistore.Client, limit int, window time.Duration) gin.HandlerFunc {
	return httpmw.NewUserRateLimiter(client, limit, window,
		httpmw.WithKeyPrefix("ratelimit:apikey"),
		httpmw.WithKeyFunc(func(c *gin.Context) string {
			if id, ok := c.Request.Context().Value(ctxkeys.APIKeyID{}).(uint64); ok {
				return fmt.Sprintf("apikey:%d", id)
			}
			return ""
		}),
	).Middleware()
}
