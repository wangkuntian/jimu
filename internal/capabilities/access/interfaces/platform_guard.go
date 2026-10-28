package interfaces

import (
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// RequirePlatformScope 将访问控制管理端点限制为平台管理员。
func RequirePlatformScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tenant.IsPlatformScope(c.Request.Context()) {
			response.Fail(c, errors.New(errors.CodeForbidden, "platform administrator required"))
			c.Abort()
			return
		}
		c.Next()
	}
}
