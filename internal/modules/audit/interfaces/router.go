package interfaces

import (
	"jimu/internal/modules/audit/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterAuditRoutes(r *gin.RouterGroup, service *application.AuditService) {
	handler := NewAuditHandler(service)
	audits := r.Group("/audits")
	{
		audits.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		// 必须在 /:id 之前注册，避免路径冲突
		audits.GET("/verify", handler.Verify)
		audits.GET("/export", handler.Export)
		audits.GET("/:id", handler.Get)
	}
}
