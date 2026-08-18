package interfaces

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/role/application"
	"jimu/internal/modules/role/domain"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func TestRoleCreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/roles", func(c *gin.Context) {
		c.Set("validated_req", &application.CreateRoleRequest{Name: "admin", Description: "ops"})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Create(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles", strings.NewReader(`{"name":"admin","description":"ops"}`)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRoleGetReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles/:id", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles/7", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRoleGetInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles/:id", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles/abc", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoleListReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles", func(c *gin.Context) {
		c.Set("validated_query", &pagination.Pagination{})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRoleListRejectsInvalidSort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles", func(c *gin.Context) {
		c.Set("validated_query", &pagination.Pagination{Sort: "password", Order: "desc"})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles?sort=password", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoleUpdateInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/roles/:id", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/roles/abc", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoleUpdateReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/roles/:id", func(c *gin.Context) {
		c.Set("validated_req", &application.UpdateRoleRequest{Name: "admin", Description: "ops"})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).Update(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/roles/7", strings.NewReader(`{"name":"admin","description":"ops"}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRoleAssignPermissionsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/roles/:id/permissions", func(c *gin.Context) {
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).AssignPermissions(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles/abc/permissions", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoleAssignPermissionsReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/roles/:id/permissions", func(c *gin.Context) {
		c.Set("validated_req", &application.AssignPermissionsRequest{PermissionIDs: []uint64{1, 2}})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).AssignPermissions(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles/7/permissions", strings.NewReader(`{"permission_ids":[1,2]}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
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

type errDeleteRoleRepository struct{ fakeRoleRepository }

func (r *errDeleteRoleRepository) Delete(context.Context, uint64) error { return errors.New("db down") }

func TestRoleDeleteServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/roles/:id", NewRoleHandler(application.NewRoleService(&errDeleteRoleRepository{})).Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/roles/7", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
