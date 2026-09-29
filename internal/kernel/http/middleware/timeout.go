package middleware

import (
	"context"
	"net/http"
	"time"

	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Timeout 请求超时中间件：为请求上下文注入 deadline，供下游 DB/Redis/RPC 提前中止。
// 若 handler 在 deadline 之后返回且未产出响应（未写 body 且状态码仍为默认 200），
// 则补 1008/504，避免客户端长时间无响应；连接级硬上限由 http.write_timeout_sec 兜底。
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if duration <= 0 {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() && c.Writer.Status() == http.StatusOK {
			response.Fail(c, apperrors.New(apperrors.CodeTimeout, "request timeout"))
		}
	}
}
