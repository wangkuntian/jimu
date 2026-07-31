package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/modules/audit/domain"

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
