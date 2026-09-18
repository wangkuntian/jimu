package interfaces

import (
	"jimu/internal/platform/feature"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminFeatureHandler Feature Flag 管理 handler
type AdminFeatureHandler struct {
	manager *feature.Manager
}

// NewAdminFeatureHandler 创建 Feature Flag 管理 handler
func NewAdminFeatureHandler(manager *feature.Manager) *AdminFeatureHandler {
	return &AdminFeatureHandler{manager: manager}
}

// List 获取所有 Feature Flag
func (h *AdminFeatureHandler) List(c *gin.Context) {
	if h.manager == nil {
		response.OK(c, []interface{}{})
		return
	}
	response.OK(c, h.manager.List())
}

// Update 更新 Feature Flag
func (h *AdminFeatureHandler) Update(c *gin.Context) {
	if h.manager == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "feature flag not initialized"))
		return
	}
	name := c.Param("name")
	var req struct {
		Enabled    *bool `json:"enabled"`
		Percentage *int  `json:"percentage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}

	updated := h.manager.Update(name, func(flag *feature.Flag) {
		if req.Enabled != nil {
			flag.Enabled = *req.Enabled
		}
		if req.Percentage != nil {
			flag.Percentage = *req.Percentage
		}
	})
	if !updated {
		response.Fail(c, errors.New(errors.CodeNotFound, "feature flag not found: "+name))
		return
	}
	response.OK(c, gin.H{"updated": name})
}
