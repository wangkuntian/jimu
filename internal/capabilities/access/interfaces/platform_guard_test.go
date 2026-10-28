package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
)

func TestRequirePlatformScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Platform") == "true" {
			c.Request = c.Request.WithContext(tenant.WithPlatformScope(c.Request.Context()))
		}
		c.Next()
	})
	r.GET("/roles", RequirePlatformScope(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	platform := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	req.Header.Set("X-Platform", "true")
	r.ServeHTTP(platform, req)
	if platform.Code != http.StatusNoContent {
		t.Fatalf("platform status = %d, want %d", platform.Code, http.StatusNoContent)
	}

	tenantAdmin := httptest.NewRecorder()
	r.ServeHTTP(tenantAdmin, httptest.NewRequest(http.MethodGet, "/roles", nil))
	if tenantAdmin.Code != http.StatusForbidden {
		t.Fatalf("tenant admin status = %d, want %d", tenantAdmin.Code, http.StatusForbidden)
	}
}
