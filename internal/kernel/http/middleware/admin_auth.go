package middleware

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员权限校验中间件
// 前置：AuthMiddleware 设置 user_id，AuthorizationMiddleware 设置 roles 并按 Casbin 校验。
// 此处检查用户是否拥有管理员角色（角色名来自 DB，seed 默认"超级管理员"）。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
			c.Abort()
			return
		}

		roles, exists := c.Get("roles")
		if !exists || roles == nil {
			response.Fail(c, errors.New(errors.CodeForbidden, "admin role required"))
			c.Abort()
			return
		}

		isAdmin := false
		if roleList, ok := roles.([]string); ok {
			for _, role := range roleList {
				if role == "超级管理员" || role == "admin" {
					isAdmin = true
					break
				}
			}
		}

		if !isAdmin {
			response.Fail(c, errors.New(errors.CodeForbidden, "admin role required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
