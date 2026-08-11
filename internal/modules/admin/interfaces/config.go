package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminConfigHandler 配置热更新 handler
type AdminConfigHandler struct {
	service *application.AdminConfigService
}

// NewAdminConfigHandler 创建配置热更新 handler
func NewAdminConfigHandler(service *application.AdminConfigService) *AdminConfigHandler {
	return &AdminConfigHandler{service: service}
}

// Get 获取所有配置
func (h *AdminConfigHandler) Get(c *gin.Context) {
	config, err := h.service.GetAllConfig(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, config)
}

// Update 更新单个配置
func (h *AdminConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	if !h.service.IsValidKey(key) {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid config key: "+key))
		return
	}
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.UpdateConfig(c.Request.Context(), key, req.Value); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"updated": key})
}

// Reload 触发配置重载：从 Redis 重读全部配置并发布事件应用
func (h *AdminConfigHandler) Reload(c *gin.Context) {
	if err := h.service.ReloadConfig(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"reloaded": true})
}
