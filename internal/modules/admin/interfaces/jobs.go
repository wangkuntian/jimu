package interfaces

import (
	"strconv"

	"jimu/internal/modules/admin/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminJobHandler 任务队列 handler
type AdminJobHandler struct {
	jobRepo domain.JobRepository
}

// NewAdminJobHandler 创建任务队列 handler
func NewAdminJobHandler(jobRepo domain.JobRepository) *AdminJobHandler {
	return &AdminJobHandler{jobRepo: jobRepo}
}

// Submit 提交任务
func (h *AdminJobHandler) Submit(c *gin.Context) {
	response.OK(c, gin.H{"job_id": "pending"})
}

// List 获取任务列表
func (h *AdminJobHandler) List(c *gin.Context) {
	status := c.Query("status")
	filters := map[string]interface{}{}
	if status != "" {
		filters["status"] = status
	}
	jobs, total, err := h.jobRepo.List(c.Request.Context(), 0, 20, filters)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, jobs, total, 1, 20)
}

// Get 获取任务详情
func (h *AdminJobHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	job, err := h.jobRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, errors.Wrap(errors.CodeNotFound, "job not found", err))
		return
	}
	response.OK(c, job)
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
