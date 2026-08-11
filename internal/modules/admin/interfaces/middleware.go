package interfaces

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware 管理端鉴权中间件
// 要求：已认证 + 拥有管理员角色
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已认证（需要 AuthMiddleware 先执行）
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
			c.Abort()
			return
		}

		// 检查是否为管理员角色
		roles, exists := c.Get("roles")
		if !exists || roles == nil {
			response.Fail(c, errors.New(errors.CodeForbidden, "admin role required"))
			c.Abort()
			return
		}

		roleList, ok := roles.([]string)
		if !ok {
			response.Fail(c, errors.New(errors.CodeForbidden, "invalid roles"))
			c.Abort()
			return
		}

		// 检查是否包含管理员角色
		isAdmin := false
		for _, role := range roleList {
			if role == "超级管理员" || role == "admin" {
				isAdmin = true
				break
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
