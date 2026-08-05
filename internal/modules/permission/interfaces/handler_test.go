package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/permission/application"
	"jimu/internal/modules/role/domain"

	"github.com/gin-gonic/gin"
)

func TestPermissionCreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/permissions", func(c *gin.Context) {
		c.Set("validated_req", &application.CreatePermissionRequest{Name: "user:read", Resource: "user", Action: "read"})
		NewPermissionHandler(application.NewPermissionService(&fakePermissionRepository{})).Create(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/permissions", strings.NewReader(`{"name":"user:read","resource":"user","action":"read"}`)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestPermissionDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/permissions/:id", NewPermissionHandler(application.NewPermissionService(&fakePermissionRepository{})).Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/permissions/7", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

type fakePermissionRepository struct{}

func (r *fakePermissionRepository) FindByID(context.Context, uint64) (*domain.Permission, error) {
	return &domain.Permission{ID: 7, Name: "user:read"}, nil
}
func (r *fakePermissionRepository) List(context.Context, int, int, string, string) ([]domain.Permission, int64, error) {
	return nil, 0, nil
}
func (r *fakePermissionRepository) Create(_ context.Context, permission *domain.Permission) error {
	permission.ID = 7
	return nil
}
func (r *fakePermissionRepository) Update(context.Context, *domain.Permission) error { return nil }
func (r *fakePermissionRepository) Delete(context.Context, uint64) error             { return nil }
