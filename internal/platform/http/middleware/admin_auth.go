package middleware

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员权限校验中间件
// 前置：AuthMiddleware 设置 user_id，AuthorizationMiddleware 设置 roles 并按 Casbin 校验。
// 此处依赖 AuthorizationMiddleware 基于角色 scope 写入的平台范围标记。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
			c.Abort()
			return
		}

		platform, _ := c.Get("platform_scope")
		if enabled, _ := platform.(bool); !enabled {
			response.Fail(c, errors.New(errors.CodeForbidden, "platform administrator required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
