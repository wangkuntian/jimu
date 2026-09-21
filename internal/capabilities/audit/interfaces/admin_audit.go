package interfaces

import (
	"strconv"

	auditdomain "jimu/internal/capabilities/audit/domain"
	auditinfra "jimu/internal/capabilities/audit/infrastructure"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/pagination"
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
	return &AdminAuditHandler{repo: auditinfra.NewMysqlAuditRepository(db, "")}
}

// List 获取审计日志列表（按上下文租户过滤；0=平台级视角不过滤）
func (h *AdminAuditHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	logs, total, err := h.repo.List(c.Request.Context(), tenant.FromContext(c.Request.Context()), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, logs, total, p.Page, p.PageSize)
}

// paginationFromQuery 从 query 解析分页参数（管理端列表端点共用）。
func paginationFromQuery(c *gin.Context) pagination.Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sort := c.DefaultQuery("sort", "id")
	order := c.DefaultQuery("order", "desc")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return pagination.Pagination{Page: page, PageSize: pageSize, Sort: sort, Order: order}
}
