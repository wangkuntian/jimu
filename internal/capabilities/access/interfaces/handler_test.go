package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		c.Set("validated_query", &roleListQuery{})
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
		c.Set("validated_query", &roleListQuery{Pagination: pagination.Pagination{Sort: "password", Order: "desc"}})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles?sort=password", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRoleListPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	status := int8(0)
	repo := &fakeRoleRepository{}
	r := gin.New()
	r.GET("/roles", func(c *gin.Context) {
		c.Set("validated_query", &roleListQuery{
			Pagination: pagination.Pagination{Page: 1, PageSize: 20},
			TenantID:   "9",
			Keyword:    "ops",
			Status:     &status,
		})
		NewRoleHandler(application.NewRoleService(repo)).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles?tenant_id=9&keyword=ops&status=0", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if repo.listFilter.TenantID != 9 || repo.listFilter.Keyword != "ops" || repo.listFilter.Status == nil || *repo.listFilter.Status != 0 {
		t.Fatalf("filter = %#v", repo.listFilter)
	}
}

func TestRoleListRejectsZeroTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles", func(c *gin.Context) {
		c.Set("validated_query", &roleListQuery{
			Pagination: pagination.Pagination{Page: 1, PageSize: 20},
			TenantID:   "0",
		})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles?tenant_id=0", nil))

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
		ids := []string{"1", "2"}
		c.Set("validated_req", &application.AssignPermissionsRequest{PermissionIDs: &ids})
		NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).AssignPermissions(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles/7/permissions", strings.NewReader(`{"permission_ids":["1","2"]}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRoleAssignPermissionsAllowsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRoleRepository{}
	r := gin.New()
	r.POST("/roles/:id/permissions",
		middleware.ValidateJSON(&application.AssignPermissionsRequest{}),
		NewRoleHandler(application.NewRoleService(repo)).AssignPermissions)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/roles/7/permissions", strings.NewReader(`{"permission_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if repo.assigned == nil || len(repo.assigned) != 0 {
		t.Fatalf("assigned = %#v, want non-nil empty slice", repo.assigned)
	}
}

func TestRoleAssignPermissionsDeduplicatesIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ids := []string{"2", "1", "2"}
	repo := &fakeRoleRepository{}
	r := gin.New()
	r.POST("/roles/:id/permissions", func(c *gin.Context) {
		c.Set("validated_req", &application.AssignPermissionsRequest{PermissionIDs: &ids})
		NewRoleHandler(application.NewRoleService(repo)).AssignPermissions(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/roles/7/permissions", strings.NewReader(`{"permission_ids":["2","1","2"]}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(repo.assigned) != 2 || repo.assigned[0] != 2 || repo.assigned[1] != 1 {
		t.Fatalf("assigned = %v", repo.assigned)
	}
}

func TestRoleGetPermissionsReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/roles/:id/permissions", NewRoleHandler(application.NewRoleService(&fakeRoleRepository{})).GetPermissions)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles/7/permissions", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body struct {
		Data []application.PermissionResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data == nil || len(body.Data) != 0 {
		t.Fatalf("data = %#v, want empty array", body.Data)
	}
}

func TestRoleOptionsReturnsTenantRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRoleRepository{options: []domain.Role{{
		ID:       8,
		TenantID: 9,
		Name:     "租户管理员",
		Code:     contract.RoleCodeTenantAdmin,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
	}}}
	r := gin.New()
	r.GET("/roles/options", func(c *gin.Context) {
		c.Set("validated_query", &roleOptionQuery{})
		c.Request = c.Request.WithContext(tenant.WithTenant(c.Request.Context(), 9))
		NewRoleHandler(application.NewRoleService(repo)).Options(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/roles/options", nil))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []application.RoleOptionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, contract.RoleCodeTenantAdmin, body.Data[0].Code)
	assert.Equal(t, "8", body.Data[0].ID)
}

func TestRoleOptionsRouteIsNotPlatformGuarded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRoleRepository{options: []domain.Role{{
		ID:       8,
		TenantID: 9,
		Name:     "租户管理员",
		Code:     contract.RoleCodeTenantAdmin,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
	}}}
	r := gin.New()
	rg := r.Group("/api/v1")
	rg.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(tenant.WithTenant(c.Request.Context(), 9))
		c.Next()
	})
	RegisterRoleRoutes(rg, application.NewRoleService(repo))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/roles/options", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
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

type fakeRoleRepository struct {
	role        *domain.Role
	permissions []domain.Permission
	listFilter  domain.RoleListFilter
	assigned    []uint64
	options     []domain.Role
}

func (r *fakeRoleRepository) FindByID(context.Context, uint64) (*domain.Role, error) {
	if r.role != nil {
		return r.role, nil
	}
	return &domain.Role{ID: 7, Name: "admin"}, nil
}
func (r *fakeRoleRepository) List(_ context.Context, filter domain.RoleListFilter, _ int, _ int, _ string, _ string) ([]domain.Role, int64, error) {
	r.listFilter = filter
	return nil, 0, nil
}
func (r *fakeRoleRepository) Create(_ context.Context, role *domain.Role) error {
	role.ID = 7
	return nil
}
func (r *fakeRoleRepository) Update(context.Context, *domain.Role) error { return nil }
func (r *fakeRoleRepository) AssignPermissions(_ context.Context, _ uint64, permissionIDs []uint64) error {
	r.assigned = append([]uint64(nil), permissionIDs...)
	if permissionIDs != nil && len(permissionIDs) == 0 {
		r.assigned = []uint64{}
	}
	return nil
}
func (r *fakeRoleRepository) GetPermissions(context.Context, uint64) ([]domain.Permission, error) {
	return r.permissions, nil
}

func (r *fakeRoleRepository) ListOptions(context.Context, uint64) ([]domain.Role, error) {
	return r.options, nil
}

func (r *fakeRoleRepository) FindPermissionsByIDs(_ context.Context, ids []uint64) ([]domain.Permission, error) {
	out := make([]domain.Permission, 0, len(ids))
	for _, id := range ids {
		out = append(out, domain.Permission{ID: id, Resource: "/api/v1/test", Action: "GET"})
	}
	return out, nil
}

type errDeleteRoleRepository struct{ fakeRoleRepository }

func (r *errDeleteRoleRepository) Update(context.Context, *domain.Role) error {
	return errors.New("db down")
}

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
