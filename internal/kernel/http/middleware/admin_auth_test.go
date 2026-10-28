package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func adminAuthRouter(platform bool, setPlatform bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		if setPlatform {
			c.Set("platform_scope", platform)
		}
		c.Next()
	})
	r.Use(AdminAuth())
	r.GET("/admin/status", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAdminAuthAllowsAdminRole(t *testing.T) {
	r := adminAuthRouter(true, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthAllowsPlatformSuperAdminRole(t *testing.T) {
	r := adminAuthRouter(true, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthAllowsEnglishAdminRole(t *testing.T) {
	r := adminAuthRouter(true, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthRejectsNonAdminRole(t *testing.T) {
	r := adminAuthRouter(false, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminAuthRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AdminAuth())
	r.GET("/admin/status", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuthRequiresRoles(t *testing.T) {
	r := adminAuthRouter(false, false)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
