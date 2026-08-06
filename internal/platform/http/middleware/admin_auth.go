package middleware

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员权限校验中间件
// 检查用户是否拥有 admin scope
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 context 获取用户 scopes（由 JWT 中间件设置）
		scopes, ok := c.Get("scopes")
		if !ok {
			response.Fail(c, errors.New(errors.CodeForbidden, "admin access required"))
			c.Abort()
			return
		}

		// 检查是否有 admin scope
		hasAdmin := false
		if scopeList, ok := scopes.([]string); ok {
			for _, s := range scopeList {
				if s == "admin" || s == "super_admin" {
					hasAdmin = true
					break
				}
			}
		}

		if !hasAdmin {
			response.Fail(c, errors.New(errors.CodeForbidden, "admin access required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
