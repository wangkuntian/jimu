package interfaces

import (
	"strconv"

	"jimu/internal/modules/role/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *application.RoleService
}

func NewRoleHandler(service *application.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req application.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	role, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, role)
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, role)
}

func (h *RoleHandler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := p.Normalize("id", "name", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	roles, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, roles, total, p.Page, p.PageSize)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.UpdateRoleRequest
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

func (h *RoleHandler) Delete(c *gin.Context) {
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

func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.AssignPermissions(c.Request.Context(), id, req.PermissionIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
