package interfaces

import (
	"jimu/internal/modules/audit/application"

	"github.com/gin-gonic/gin"
)

func RegisterAuditRoutes(r *gin.RouterGroup, service *application.AuditService) {
	handler := NewAuditHandler(service)
	audits := r.Group("/audits")
	{
		audits.GET("", handler.List)
		audits.GET("/:id", handler.Get)
	}
}
