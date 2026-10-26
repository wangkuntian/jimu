package interfaces

import (
	"jimu/internal/capabilities/access/application"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterRoleRoutes(r *gin.RouterGroup, service *application.RoleService) {
	handler := NewRoleHandler(service)
	options := r.Group("/roles")
	options.GET("/options", middleware.ValidateQuery(&roleOptionQuery{}), handler.Options)

	roles := r.Group("/roles")
	roles.Use(RequirePlatformScope())
	{
		roles.POST("", middleware.ValidateJSON(&application.CreateRoleRequest{}), handler.Create)
		roles.GET("", middleware.ValidateQuery(&roleListQuery{}), handler.List)
		roles.GET("/:id", handler.Get)
		roles.PUT("/:id", middleware.ValidateJSON(&application.UpdateRoleRequest{}), handler.Update)
		roles.DELETE("/:id", handler.Delete)
		roles.GET("/:id/permissions", handler.GetPermissions)
		roles.POST("/:id/permissions", middleware.ValidateJSON(&application.AssignPermissionsRequest{}), handler.AssignPermissions)
	}
}

type roleListQuery struct {
	pagination.Pagination
	TenantID string `form:"tenant_id"`
	Keyword  string `form:"keyword"`
	Status   *int8  `form:"status" binding:"omitempty,oneof=0 1"`
}

type roleOptionQuery struct {
	TenantID string `form:"tenant_id"`
}
