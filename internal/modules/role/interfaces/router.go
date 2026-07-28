package interfaces

import (
	"jimu/internal/modules/role/application"

	"github.com/gin-gonic/gin"
)

func RegisterRoleRoutes(r *gin.RouterGroup, service *application.RoleService) {
	handler := NewRoleHandler(service)
	roles := r.Group("/roles")
	{
		roles.POST("", handler.Create)
		roles.GET("", handler.List)
		roles.GET("/:id", handler.Get)
		roles.PUT("/:id", handler.Update)
		roles.DELETE("/:id", handler.Delete)
		roles.POST("/:id/permissions", handler.AssignPermissions)
	}
}
