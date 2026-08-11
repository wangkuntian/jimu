package interfaces

import (
	auditdomain "jimu/internal/modules/audit/domain"
	auditinfra "jimu/internal/modules/audit/infrastructure"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminAuditHandler 审计日志 handler
// 复用 audit 模块仓储，保证与 audit_logs 表实际 schema（006 迁移）一致
type AdminAuditHandler struct {
	repo auditdomain.AuditRepository
}

// NewAdminAuditHandler 创建审计日志 handler
func NewAdminAuditHandler(db *gorm.DB) *AdminAuditHandler {
	return &AdminAuditHandler{repo: auditinfra.NewMysqlAuditRepository(db)}
}

// List 获取审计日志列表
func (h *AdminAuditHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	logs, total, err := h.repo.List(c.Request.Context(), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, logs, total, p.Page, p.PageSize)
}
