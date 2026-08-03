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
