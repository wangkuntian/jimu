package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/platform/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册管理端路由
// signingSecret 用于管理 API 的 HMAC 请求签名验证
func RegisterRoutes(r *gin.RouterGroup, service *application.Service, signingSecret ...string) {
	handler := NewHandler(service)
	admin := r.Group("/admin")
	{
		// 错误码文档（公开访问，便于前端/第三方查阅）
		admin.GET("/error-codes", handler.GetErrorCodes)

		// 以下接口需要管理员权限 + 请求签名验证（防篡改 + 防重放）
		secret := "change-me-in-production"
		if len(signingSecret) > 0 && signingSecret[0] != "" {
			secret = signingSecret[0]
		}
		authorized := admin.Group("")
		authorized.Use(AdminAuthMiddleware())
		authorized.Use(middleware.Signature(middleware.DefaultSignatureConfig([]byte(secret))))
		{
			authorized.GET("/status", handler.GetStatus)
			authorized.GET("/users/online", handler.GetOnlineUsers)
			authorized.POST("/users/:user_id/logout", handler.ForceLogout)
		}
	}
}
