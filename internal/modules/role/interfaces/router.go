package interfaces

import (
	"jimu/internal/modules/role/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterRoleRoutes(r *gin.RouterGroup, service *application.RoleService) {
	handler := NewRoleHandler(service)
	roles := r.Group("/roles")
	{
		roles.POST("", middleware.ValidateJSON(&application.CreateRoleRequest{}), handler.Create)
		roles.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		roles.GET("/:id", handler.Get)
		roles.PUT("/:id", middleware.ValidateJSON(&application.UpdateRoleRequest{}), handler.Update)
		roles.DELETE("/:id", handler.Delete)
		roles.POST("/:id/permissions", middleware.ValidateJSON(&application.AssignPermissionsRequest{}), handler.AssignPermissions)
	}
}
