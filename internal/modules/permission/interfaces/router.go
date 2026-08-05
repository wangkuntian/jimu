package interfaces

import (
	"jimu/internal/modules/permission/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterPermissionRoutes(r *gin.RouterGroup, service *application.PermissionService) {
	handler := NewPermissionHandler(service)
	perms := r.Group("/permissions")
	{
		perms.POST("", middleware.ValidateJSON(&application.CreatePermissionRequest{}), handler.Create)
		perms.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		perms.GET("/:id", handler.Get)
		perms.PUT("/:id", middleware.ValidateJSON(&application.UpdatePermissionRequest{}), handler.Update)
		perms.DELETE("/:id", handler.Delete)
	}
}
