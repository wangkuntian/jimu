package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/role/application"
	"jimu/internal/modules/role/domain"

	"github.com/gin-gonic/gin"
)

func TestRoleCreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/roles", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles", strings.NewReader(`{"name":"admin","description":"ops"}`)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRoleDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/roles/:id", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/roles/7", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

type fakeRoleRepository struct{}

func (r *fakeRoleRepository) FindByID(context.Context, uint64) (*domain.Role, error) {
	return &domain.Role{ID: 7, Name: "admin"}, nil
}
func (r *fakeRoleRepository) List(context.Context, int, int, string, string) ([]domain.Role, int64, error) {
	return nil, 0, nil
}
func (r *fakeRoleRepository) Create(_ context.Context, role *domain.Role) error {
	role.ID = 7
	return nil
}
func (r *fakeRoleRepository) Update(context.Context, *domain.Role) error { return nil }
func (r *fakeRoleRepository) Delete(context.Context, uint64) error       { return nil }
func (r *fakeRoleRepository) AssignPermissions(context.Context, uint64, []uint64) error {
	return nil
}
func (r *fakeRoleRepository) GetPermissions(context.Context, uint64) ([]domain.Permission, error) {
	return nil, nil
}
