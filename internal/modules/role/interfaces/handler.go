package interfaces

import (
	"strconv"

	"jimu/internal/modules/role/application"
	"jimu/internal/shared/errors"
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
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	role, err := h.service.Create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, role)
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
	roles, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, roles)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.Update(c.Request.Context(), id, req.Name, req.Description); err != nil {
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
	response.OK(c, nil)
}

func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req struct {
		PermissionIDs []uint64 `json:"permission_ids" binding:"required"`
	}
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
