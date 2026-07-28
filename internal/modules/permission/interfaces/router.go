package interfaces

import (
	"jimu/internal/modules/permission/application"

	"github.com/gin-gonic/gin"
)

func RegisterPermissionRoutes(r *gin.RouterGroup, service *application.PermissionService) {
	handler := NewPermissionHandler(service)
	perms := r.Group("/permissions")
	{
		perms.POST("", handler.Create)
		perms.GET("", handler.List)
		perms.GET("/:id", handler.Get)
		perms.PUT("/:id", handler.Update)
		perms.DELETE("/:id", handler.Delete)
	}
}
