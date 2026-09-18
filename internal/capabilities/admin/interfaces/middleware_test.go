package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	run := func(setup func(c *gin.Context)) int {
		r := gin.New()
		// setup 中间件先于 AdminAuthMiddleware 注入认证上下文
		if setup != nil {
			r.Use(func(c *gin.Context) {
				setup(c)
				c.Next()
			})
		}
		r.Use(AdminAuthMiddleware())
		r.GET("/x", func(c *gin.Context) {
			c.Status(200)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		return w.Code
	}

	// 未认证
	assert.Equal(t, http.StatusUnauthorized, run(nil))

	// 已认证但无角色
	assert.Equal(t, http.StatusForbidden, run(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
	}))

	// 角色类型错误
	assert.Equal(t, http.StatusForbidden, run(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("roles", "admin")
	}))

	// 角色不含管理员
	assert.Equal(t, http.StatusForbidden, run(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("roles", []string{"member"})
	}))

	// 管理员角色
	assert.Equal(t, http.StatusOK, run(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("roles", []string{"member", "admin"})
	}))

	// 超级管理员角色
	assert.Equal(t, http.StatusOK, run(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("roles", []string{"超级管理员"})
	}))
}
