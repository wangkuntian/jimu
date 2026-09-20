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

type fakeAuthzStore struct{}

func (s *fakeAuthzStore) RolesForUser(context.Context, uint64) ([]string, error) {
	return []string{"admin"}, nil
}

func (s *fakeAuthzStore) Policies(context.Context) ([]access.Policy, error) {
	return []access.Policy{{Role: "admin", Resource: "/api/v1/users", Action: http.MethodGet}}, nil
}
