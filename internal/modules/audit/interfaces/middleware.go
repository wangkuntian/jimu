package interfaces

import (
	"bytes"
	"io"
	"time"

	"jimu/internal/modules/audit/application"
	"jimu/internal/modules/audit/domain"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware 自动记录 API 审计日志
func AuditMiddleware(auditService *application.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过健康检查和文档
		path := c.Request.URL.Path
		if path == "/health" || path == "/swagger/" || path == "/debug/" {
			c.Next()
			return
		}

		start := time.Now()

		// 读取请求体（可选）
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		c.Next()

		// 记录审计日志
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		auditLog := domain.AuditLog{
			UserID:   userID.(uint64),
			Username: username.(string),
			Action:   c.Request.Method,
			Resource: path,
			IP:       c.ClientIP(),
			Method:   c.Request.Method,
			Path:     path,
			Status:   c.Writer.Status(),
			Detail:   time.Since(start).String(),
		}

		// 异步记录，不影响主流程
		go auditService.Record(c.Request.Context(), auditLog)
	}
}
