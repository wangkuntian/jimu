package interfaces

import (
	"strings"
	"time"

	"jimu/internal/modules/audit/domain"

	"github.com/gin-gonic/gin"
)

type Queue interface {
	Enqueue(domain.AuditLog) bool
}

func AuditMiddleware(queue Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if isManagementPath(path) {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		queue.Enqueue(domain.AuditLog{
			UserID:   optionalUint64(c, "user_id"),
			Username: optionalString(c, "username"),
			Action:   c.Request.Method,
			Resource: path,
			IP:       c.ClientIP(),
			Method:   c.Request.Method,
			Path:     path,
			Status:   c.Writer.Status(),
			Detail:   time.Since(start).String(),
		})
	}
}

func isManagementPath(path string) bool {
	return path == "/livez" || path == "/readyz" || path == "/metrics" ||
		strings.HasPrefix(path, "/swagger/") || strings.HasPrefix(path, "/debug/")
}

func optionalUint64(c *gin.Context, key string) uint64 {
	value, ok := c.Get(key)
	if !ok {
		return 0
	}
	result, _ := value.(uint64)
	return result
}

func optionalString(c *gin.Context, key string) string {
	value, ok := c.Get(key)
	if !ok {
		return ""
	}
	result, _ := value.(string)
	return result
}
