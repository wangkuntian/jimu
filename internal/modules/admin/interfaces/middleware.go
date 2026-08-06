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

// AdminOrOwnerMiddleware 管理员或资源所有者权限
// 允许管理员或资源所有者本人访问
func AdminOrOwnerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已认证
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
			c.Abort()
			return
		}

		// 检查是否为管理员
		if roles, exists := c.Get("roles"); exists {
			if roleList, ok := roles.([]string); ok {
				for _, role := range roleList {
					if role == "超级管理员" || role == "admin" {
						c.Next()
						return
					}
				}
			}
		}

		// 非管理员只能操作自己的资源
		if targetID := c.Param("user_id"); targetID != "" {
			if uid, ok := userID.(uint64); ok {
				if targetUint, err := parseUint(targetID); err == nil && uid == targetUint {
					c.Next()
					return
				}
			}
		}

		response.Fail(c, errors.New(errors.CodeForbidden, "permission denied"))
		c.Abort()
	}
}

func parseUint(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, ErrInvalidID
		}
		n = n*10 + uint64(c-'0')
	}
	return n, nil
}

var ErrInvalidID = errors.New(errors.CodeInvalidParam, "invalid id")
