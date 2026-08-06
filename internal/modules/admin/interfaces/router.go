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
		admin.GET("/status", handler.GetStatus)
		admin.GET("/users/online", handler.GetOnlineUsers)
		admin.POST("/users/:user_id/logout", handler.ForceLogout)
	}
}
