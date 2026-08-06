package auth

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

func PermissionMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			response.Fail(c, errors.New(errors.CodeForbidden, "no roles assigned"))
			c.Abort()
			return
		}

		roleList, ok := roles.([]string)
		if !ok {
			response.Fail(c, errors.New(errors.CodeForbidden, "invalid roles"))
			c.Abort()
			return
		}

		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed := false
		for _, role := range roleList {
			if ok, _ := enforcer.Enforce(role, obj, act); ok {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Fail(c, errors.New(errors.CodeForbidden, "permission denied"))
			c.Abort()
			return
		}
		c.Next()
	}
}
