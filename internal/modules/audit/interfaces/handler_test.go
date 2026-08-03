package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/audit/application"
	"jimu/internal/modules/audit/domain"

	"github.com/gin-gonic/gin"
)

func TestAuditListInvalidQueryReturnsStableBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", NewAuditHandler(nil).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit?sort=password", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if strings.Contains(w.Body.String(), "gin.Error") {
		t.Fatalf("leaked gin error: %s", w.Body.String())
	}
}

func TestAuditGetInvalidIDReturnsStableBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/:id", NewAuditHandler(nil).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/not-number", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuditGetReturnsLogDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/:id", NewAuditHandler(application.NewAuditService(&fakeAuditRepository{})).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/7", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]any)
	if data["username"] != "alice" || data["action"] != "create" {
		t.Fatalf("body = %#v", body)
	}
}

type fakeAuditRepository struct{}

func (r *fakeAuditRepository) Create(context.Context, *domain.AuditLog) error { return nil }
func (r *fakeAuditRepository) CreateBatch(context.Context, []domain.AuditLog) error {
	return nil
}
func (r *fakeAuditRepository) FindByID(context.Context, uint64) (*domain.AuditLog, error) {
	return &domain.AuditLog{ID: 7, Username: "alice", Action: "create"}, nil
}
func (r *fakeAuditRepository) List(context.Context, int, int, string, string) ([]domain.AuditLog, int64, error) {
	return nil, 0, nil
}
