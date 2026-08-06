package middleware

import (
	"bytes"
	"encoding/json"
	"time"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// IdempotencyMiddleware 幂等性中间件
// 客户端在请求头中携带 Idempotency-Key: <uuid>
// 相同 key 的重复请求会返回缓存的响应，不会重复执行业务逻辑
func IdempotencyMiddleware(redis *redis.Client, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		// 验证 key 格式（简单长度检查）
		if len(key) < 8 || len(key) > 128 {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid idempotency key format"))
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		cacheKey := "idempotency:" + key

		// 检查是否已存在
		cached, err := redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			// 命中缓存，返回之前的响应
			var cachedResp cachedResponse
			if err := json.Unmarshal([]byte(cached), &cachedResp); err == nil {
				c.Status(cachedResp.Status)
				for k, v := range cachedResp.Headers {
					c.Header(k, v)
				}
				c.Writer.Write([]byte(cachedResp.Body))
				c.Abort()
				return
			}
		}

		// 未命中，执行业务逻辑并缓存响应
		w := newResponseBodyWriter(c.Writer, 8192)
		c.Writer = w

		c.Next()

		// 只缓存成功的响应（2xx）
		status := c.Writer.Status()
		if status >= 200 && status < 300 {
			cached := cachedResponse{
				Status:  status,
				Headers: map[string]string{"Content-Type": w.Header().Get("Content-Type")},
				Body:    w.capturedBody(),
			}
			if data, err := json.Marshal(cached); err == nil {
				redis.Set(ctx, cacheKey, data, ttl)
			}
		}
	}
}

// cachedResponse 缓存的响应
type cachedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// 确保 responseBodyWriter 的方法可用
var _ = bytes.NewBuffer
