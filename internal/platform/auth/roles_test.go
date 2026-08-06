package auth

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

func TestAuthorizationMiddlewareAllowsRolePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := testEnforcer(t)
	store := &fakeAuthorizationStore{
		roles:    []string{"admin"},
		policies: []Policy{{Role: "admin", Resource: "/api/v1/users", Action: http.MethodGet}},
	}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(7))
		c.Next()
	})
	r.Use(AuthorizationMiddleware(store, enforcer))
	r.GET("/api/v1/users", func(c *gin.Context) {
		if roles, _ := c.Get("roles"); len(roles.([]string)) != 1 {
			t.Fatalf("roles = %#v", roles)
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddlewareRejectsMissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(7))
		c.Next()
	})
	r.Use(AuthorizationMiddleware(&fakeAuthorizationStore{}, testEnforcer(t)))
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAuthorizationMiddlewareHidesStoreErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(7))
		c.Next()
	})
	r.Use(AuthorizationMiddleware(&fakeAuthorizationStore{rolesErr: stderrors.New("sql password=secret")}, testEnforcer(t)))
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if got := w.Body.String(); got == "" || stderrors.Is(stderrors.New(got), stderrors.New("sql password=secret")) {
		t.Fatalf("body = %s", got)
	}
}

func testEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	enforcer, err := NewPathEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	return enforcer
}

type fakeAuthorizationStore struct {
	roles       []string
	policies    []Policy
	rolesErr    error
	policiesErr error
}

func (s *fakeAuthorizationStore) RolesForUser(context.Context, uint64) ([]string, error) {
	return s.roles, s.rolesErr
}

func (s *fakeAuthorizationStore) Policies(context.Context) ([]Policy, error) {
	return s.policies, s.policiesErr
}
