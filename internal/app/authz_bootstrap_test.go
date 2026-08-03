package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
)

func TestBusinessRoutesRequireProtectedMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	if err := registerHTTP(router, nil, fakeAuthzModule{}, fakeBusinessModule{}); err != nil {
		t.Fatal(err)
	}

	auth := httptest.NewRecorder()
	router.ServeHTTP(auth, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	if auth.Code != http.StatusNoContent {
		t.Fatalf("auth status = %d, want %d", auth.Code, http.StatusNoContent)
	}

	users := httptest.NewRecorder()
	router.ServeHTTP(users, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if users.Code != http.StatusUnauthorized {
		t.Fatalf("users status = %d, want %d", users.Code, http.StatusUnauthorized)
	}
}

type fakeAuthzModule struct{}

func (fakeAuthzModule) Name() string { return "auth" }
func (fakeAuthzModule) RegisterHTTP(r contract.Router) {
	r.Group("/api/v1").POST("/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}
func (fakeAuthzModule) RegisterJobs(contract.JobRegistry) {}
func (fakeAuthzModule) RegisterEvents(contract.EventBus)  {}
func (fakeAuthzModule) HTTPMiddleware() []gin.HandlerFunc { return nil }
func (fakeAuthzModule) Components() []contract.Component  { return nil }
func (fakeAuthzModule) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	return []gin.HandlerFunc{func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}}, nil
}

type fakeBusinessModule struct{}

func (fakeBusinessModule) Name() string { return "user" }
func (fakeBusinessModule) RegisterHTTP(r contract.Router) {
	r.Group("/api/v1").GET("/users", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}
func (fakeBusinessModule) RegisterJobs(contract.JobRegistry) {}
func (fakeBusinessModule) RegisterEvents(contract.EventBus)  {}
