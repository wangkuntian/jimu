package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAuditHandler 审计日志 handler
type AdminAuditHandler struct {
	service *application.AdminAuditService
}

// NewAdminAuditHandler 创建审计日志 handler
func NewAdminAuditHandler(service *application.AdminAuditService) *AdminAuditHandler {
	return &AdminAuditHandler{service: service}
}

// List 获取审计日志列表
func (h *AdminAuditHandler) List(c *gin.Context) {
	logs, total, err := h.service.ListAuditLogs(c.Request.Context(), 0, 20, nil)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, logs, total, 1, 20)
}
