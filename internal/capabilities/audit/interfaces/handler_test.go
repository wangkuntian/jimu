package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/audit/application"
	"jimu/internal/capabilities/audit/domain"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuditListInvalidQueryReturnsStableBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", func(c *gin.Context) {
		// 传入无效 sort 字段，触发 Normalize 失败
		c.Set("validated_query", &pagination.Pagination{Sort: "password", Order: "desc"})
		NewAuditHandler(application.NewAuditService(&fakeAuditRepository{}, "")).List(c)
	})
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
	r.GET("/audit/:id", NewAuditHandler(application.NewAuditService(&fakeAuditRepository{}, "")).Get)
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
func (r *fakeAuditRepository) List(context.Context, uint64, int, int, string, string) ([]domain.AuditLog, int64, error) {
	return nil, 0, nil
}

func (r *fakeAuditRepository) ListForVerify(context.Context, uint64, uint64, uint64, int) ([]domain.AuditLog, error) {
	return nil, nil
}

func (r *fakeAuditRepository) ChainHead(context.Context, uint64) (string, error) { return "", nil }

func (r *fakeAuditRepository) CountRange(context.Context, uint64, time.Time, time.Time) (int64, error) {
	return 0, nil
}

func (r *fakeAuditRepository) ListRange(context.Context, uint64, time.Time, time.Time, int, int) ([]domain.AuditLog, error) {
	return nil, nil
}

func TestAuditHandlerVerify(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/verify", NewAuditHandler(application.NewAuditService(&fakeAuditRepository{}, "s3cret")).Verify)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/verify?from_id=1&limit=10", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "intact")
}

func TestAuditHandlerExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/export", NewAuditHandler(application.NewAuditService(&exportAuditRepository{}, "")).Export)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/export?format=csv", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Equal(t, "1", w.Header().Get("X-Export-Rows"))
	// BOM + 表头 + 一行数据
	assert.True(t, strings.HasPrefix(w.Body.String(), "\ufeffid,created_at"))
	assert.Contains(t, w.Body.String(), "alice")
}

// exportAuditRepository 只实现导出需要的方法
type exportAuditRepository struct{ fakeAuditRepository }

func (r *exportAuditRepository) CountRange(context.Context, uint64, time.Time, time.Time) (int64, error) {
	return 1, nil
}

func (r *exportAuditRepository) ListRange(_ context.Context, _ uint64, _, _ time.Time, offset, _ int) ([]domain.AuditLog, error) {
	if offset > 0 {
		return nil, nil
	}
	return []domain.AuditLog{{ID: 1, TenantID: 1, Username: "alice", Action: "login", CreatedAt: time.Unix(1700000000, 0)}}, nil
}
