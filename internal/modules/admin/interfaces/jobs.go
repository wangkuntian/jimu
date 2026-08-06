package interfaces

import (
	"strconv"

	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminJobHandler 任务队列 handler
type AdminJobHandler struct{}

// NewAdminJobHandler 创建任务队列 handler
func NewAdminJobHandler() *AdminJobHandler {
	return &AdminJobHandler{}
}

// Submit 提交任务
func (h *AdminJobHandler) Submit(c *gin.Context) {
	response.OK(c, gin.H{"job_id": "pending"})
}

// List 获取任务列表
func (h *AdminJobHandler) List(c *gin.Context) {
	response.Page(c, []interface{}{}, 0, 1, 20)
}

// Get 获取任务详情
func (h *AdminJobHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_ = id
	response.OK(c, nil)
}

// Retry 手动重试
func (h *AdminJobHandler) Retry(c *gin.Context) {
	response.OK(c, gin.H{"retried": true})
}

// ListDeadLetters 获取死信列表
func (h *AdminJobHandler) ListDeadLetters(c *gin.Context) {
	response.Page(c, []interface{}{}, 0, 1, 20)
}

// ResolveDeadLetter 处理死信
func (h *AdminJobHandler) ResolveDeadLetter(c *gin.Context) {
	response.OK(c, gin.H{"resolved": true})
}
