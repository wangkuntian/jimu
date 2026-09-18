package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func TestProtectedMiddlewareRequiresAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := platformauth.New(strings.Repeat("s", 32), "jimu", 30, 7)
	enforcer, err := platformauth.NewPathEnforcer()
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

func (s *fakeAuthzStore) Policies(context.Context) ([]platformauth.Policy, error) {
	return []platformauth.Policy{{Role: "admin", Resource: "/api/v1/users", Action: http.MethodGet}}, nil
}
