package interfaces

import (
	"strconv"

	"jimu/internal/modules/audit/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *application.AuditService
}

func NewAuditHandler(service *application.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// List godoc
// @Summary      List audit logs
// @Tags         audits
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page"
// @Param        page_size  query     int     false  "Page size"
// @Param        sort       query     string  false  "Sort field"
// @Param        order      query     string  false  "Sort order"
// @Success      200        {object}  contract.PageResponse
// @Failure      400        {object}  contract.ErrorResponse
// @Failure      500        {object}  contract.ErrorResponse
// @Router       /audits [get]
func (h *AuditHandler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := p.Normalize("id", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	logs, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, logs, total, p.Page, p.PageSize)
}

// Get godoc
// @Summary      Get audit log
// @Tags         audits
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Audit log ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      404  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /audits/{id} [get]
func (h *AuditHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	log, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, log)
}
