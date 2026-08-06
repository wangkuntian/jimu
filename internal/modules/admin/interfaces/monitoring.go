package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminMonitoringHandler 运维监控 handler
type AdminMonitoringHandler struct {
	service *application.AdminMonitoringService
}

// NewAdminMonitoringHandler 创建运维监控 handler
func NewAdminMonitoringHandler(service *application.AdminMonitoringService) *AdminMonitoringHandler {
	return &AdminMonitoringHandler{service: service}
}

// Status 获取系统状态
func (h *AdminMonitoringHandler) Status(c *gin.Context) {
	status := h.service.GetStatus()
	response.OK(c, status)
}

// Health 获取依赖健康状态
func (h *AdminMonitoringHandler) Health(c *gin.Context) {
	health := h.service.GetHealth(c.Request.Context())
	response.OK(c, health)
}

// Metrics 获取实时指标（供前端展示）
func (h *AdminMonitoringHandler) Metrics(c *gin.Context) {
	status := h.service.GetStatus()
	response.OK(c, gin.H{
		"goroutines":  status.NumGoroutine,
		"memory_mb":   status.Memory.Alloc / 1024 / 1024,
		"gc_count":    status.NumGC,
		"uptime":      status.Uptime,
	})
}
