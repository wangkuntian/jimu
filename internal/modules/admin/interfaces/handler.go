package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Handler 管理端 handler
type Handler struct {
	service *application.Service
}

// NewHandler 创建管理端 handler
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// GetErrorCodes godoc
// @Summary      获取错误码列表
// @Description  获取所有业务错误码及其对应的 HTTP 状态码映射。用于前端和第三方系统集成时参考。包含错误码、错误消息、HTTP 状态码和分类信息。
// @Tags         系统管理
// @Produce      json
// @Success      200  {object}  response.Body  "成功，返回错误码列表"
// @Router       /api/v1/admin/error-codes [get]
func (h *Handler) GetErrorCodes(c *gin.Context) {
	response.OK(c, errors.AllErrorCodes())
}
