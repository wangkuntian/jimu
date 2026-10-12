package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/capabilities/audit/domain"

	"github.com/gin-gonic/gin"
)

type fakeQueue struct {
	logs []domain.AuditLog
}

func (q *fakeQueue) Enqueue(log domain.AuditLog) bool {
	q.logs = append(q.logs, log)
	return true
}

func TestAuditMiddlewareAllowsAnonymousRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queue := &fakeQueue{}
	r := gin.New()
	r.Use(AuditMiddleware(queue))
	r.GET("/public", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public", nil))

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	if len(queue.logs) != 1 || queue.logs[0].UserID != 0 || queue.logs[0].Username != "" {
		t.Fatalf("anonymous audit = %#v", queue.logs)
	}
}

// T1.7 审计日志的租户归属**来源**：只从 gin context 的 tenant_id 读取
// （该值由 auth.AuthMiddleware / apikey.APIKeyAuthMiddleware 从已验签 claim 或已校验
// 的 Key 归属写入），落库面由 infrastructure 的集成测试覆盖。
func TestAuditMiddlewareCarriesTenantFromGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queue := &fakeQueue{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uint64(7))
		c.Next()
	})
	r.Use(AuditMiddleware(queue))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	if len(queue.logs) != 1 || queue.logs[0].TenantID != 7 {
		t.Fatalf("audit tenant = %#v, want 7", queue.logs)
	}
}
