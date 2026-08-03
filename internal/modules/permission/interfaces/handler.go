package interfaces

import (
	"strconv"

	"jimu/internal/modules/permission/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	service *application.PermissionService
}

func NewPermissionHandler(service *application.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

func (h *PermissionHandler) Create(c *gin.Context) {
	var req application.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	perm, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, perm)
}

func (h *PermissionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	perm, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, perm)
}

func (h *PermissionHandler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := p.Normalize("id", "name", "resource", "action", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	perms, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, perms, total, p.Page, p.PageSize)
}

func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.Update(c.Request.Context(), id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
