package interfaces

import (
	"strconv"

	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminAPIKeyHandler API Key 管理 handler
type AdminAPIKeyHandler struct {
	service *application.AdminAPIKeyService
}

// NewAdminAPIKeyHandler 创建 API Key 管理 handler
func NewAdminAPIKeyHandler(service *application.AdminAPIKeyService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{service: service}
}

// List 获取 API Key 列表
func (h *AdminAPIKeyHandler) List(c *gin.Context) {
	keys, total, err := h.service.ListKeys(c.Request.Context(), 0, 20)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, keys, total, 1, 20)
}

// Create 创建 API Key（返回明文，仅此一次）
func (h *AdminAPIKeyHandler) Create(c *gin.Context) {
	response.OK(c, gin.H{"key": "jimu_xxxx", "message": "store this key safely - it won't be shown again"})
}

// Get 获取 API Key 详情
func (h *AdminAPIKeyHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	key, err := h.service.GetKey(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, key)
}

// Revoke 撤销 API Key
func (h *AdminAPIKeyHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.RevokeKey(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"revoked": true})
}
