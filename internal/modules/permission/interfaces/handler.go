package interfaces

import (
	"strconv"

	"jimu/internal/modules/permission/application"
	"jimu/internal/shared/errors"
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
	var req struct {
		Name     string `json:"name" binding:"required"`
		Resource string `json:"resource" binding:"required"`
		Action   string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	perm, err := h.service.Create(c.Request.Context(), req.Name, req.Resource, req.Action)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, perm)
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
	perms, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, perms)
}

func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req struct {
		Name     string `json:"name" binding:"required"`
		Resource string `json:"resource" binding:"required"`
		Action   string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.Update(c.Request.Context(), id, req.Name, req.Resource, req.Action); err != nil {
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
	response.OK(c, nil)
}
