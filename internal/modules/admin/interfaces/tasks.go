package interfaces

import (
	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminTaskHandler 任务调度 handler
type AdminTaskHandler struct {
	service *application.AdminTaskService
}

// NewAdminTaskHandler 创建任务调度 handler
func NewAdminTaskHandler(service *application.AdminTaskService) *AdminTaskHandler {
	return &AdminTaskHandler{service: service}
}

// List 获取任务列表
func (h *AdminTaskHandler) List(c *gin.Context) {
	tasks := h.service.ListTasks()
	response.OK(c, tasks)
}

// Trigger 手动触发任务
func (h *AdminTaskHandler) Trigger(c *gin.Context) {
	taskID := c.Param("id")
	if err := h.service.TriggerTask(c.Request.Context(), taskID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"triggered": taskID})
}

// Toggle 暂停/恢复任务
func (h *AdminTaskHandler) Toggle(c *gin.Context) {
	taskID := c.Param("id")
	if err := h.service.ToggleTask(c.Request.Context(), taskID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"toggled": taskID})
}

// History 获取任务执行历史
func (h *AdminTaskHandler) History(c *gin.Context) {
	taskID := c.Param("id")
	history := h.service.GetHistory(taskID)
	response.OK(c, history)
}
