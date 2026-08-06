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

// auditChangesKey 存储变更记录的 context key
const auditChangesKey = "audit_changes"

// SetChanges 在上下文中设置字段变更记录（供 handler 调用）
func SetChanges(c *gin.Context, changes []domain.Change) {
	c.Set(auditChangesKey, changes)
}

// GetChanges 从上下文中读取字段变更记录
func GetChanges(c *gin.Context) []domain.Change {
	val, ok := c.Get(auditChangesKey)
	if !ok {
		return nil
	}
	changes, _ := val.([]domain.Change)
	return changes
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
			Changes:  GetChanges(c),
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
