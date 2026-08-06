package interfaces

import (
	"jimu/internal/modules/admin/application"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册管理端路由
func RegisterRoutes(r *gin.RouterGroup, service *application.Service) {
	handler := NewHandler(service)
	admin := r.Group("/admin")
	{
		// 错误码文档（公开访问，便于前端/第三方查阅）
		admin.GET("/error-codes", handler.GetErrorCodes)

		// 以下接口需要管理员权限
		authorized := admin.Group("")
		authorized.Use(AdminAuthMiddleware())
		{
			authorized.GET("/status", handler.GetStatus)
			authorized.GET("/users/online", handler.GetOnlineUsers)
			authorized.POST("/users/:user_id/logout", handler.ForceLogout)
		}
	}
}
