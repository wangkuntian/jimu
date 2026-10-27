package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/kernel/access"
	"jimu/internal/kernel/auth"

	"github.com/gin-gonic/gin"
)

func TestProtectedMiddlewareRequiresAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := auth.New(strings.Repeat("s", 32), "jimu", 30, 7)
	enforcer, err := access.NewPathEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(ProtectedMiddleware(jwtUtil, &fakeAuthzStore{}, enforcer)...)
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestProtectedMiddlewareRejectsInactiveTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := auth.New(strings.Repeat("s", 32), "jimu", 30, 7)
	token, err := jwtUtil.GenerateAccess(42, 7, "sid-7")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := access.NewPathEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(ProtectedMiddleware(jwtUtil, &fakeAuthzStore{}, enforcer,
		auth.StateCheckers{Tenant: fakeTenantStateChecker(false)})...)
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestProtectedMiddlewareRejectsInactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := auth.New(strings.Repeat("s", 32), "jimu", 30, 7)
	token, err := jwtUtil.GenerateAccess(42, 7, "sid-7")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := access.NewPathEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(ProtectedMiddleware(jwtUtil, &fakeAuthzStore{}, enforcer,
		auth.StateCheckers{User: fakeUserStateChecker(false)})...)
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

type fakeAuthzStore struct{}

func (s *fakeAuthzStore) RolesForUser(context.Context, uint64) ([]access.RoleIdentity, error) {
	return []access.RoleIdentity{{ID: 1, Name: "admin", Scope: "tenant"}}, nil
}

func (s *fakeAuthzStore) Policies(context.Context) ([]access.Policy, error) {
	return []access.Policy{{Role: access.RoleSubject(1), Resource: "/api/v1/users", Action: http.MethodGet}}, nil
}

type fakeTenantStateChecker bool

func (f fakeTenantStateChecker) TenantActive(context.Context, uint64) (bool, error) {
	return bool(f), nil
}

type fakeUserStateChecker bool

func (f fakeUserStateChecker) UserActive(context.Context, uint64) (bool, error) {
	return bool(f), nil
}
