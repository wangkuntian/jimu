package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func adminAuthRouter(roles []string, setRoles bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		if setRoles {
			c.Set("roles", roles)
		}
		c.Next()
	})
	r.Use(AdminAuth())
	r.GET("/admin/status", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAdminAuthAllowsAdminRole(t *testing.T) {
	r := adminAuthRouter([]string{"超级管理员"}, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthAllowsEnglishAdminRole(t *testing.T) {
	r := adminAuthRouter([]string{"admin"}, true)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthRejectsNonAdminRole(t *testing.T) {
	r := adminAuthRouter([]string{"普通用户"}, true)
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
	r := adminAuthRouter(nil, false)
	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
