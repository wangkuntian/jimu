package interfaces

import (
	"strconv"

	"jimu/internal/platform/queue/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminJobHandler 任务队列 handler
type AdminJobHandler struct {
	jobRepo  domain.JobRepository
	deadRepo domain.DeadLetterRepository
}

// NewAdminJobHandler 创建任务队列 handler
func NewAdminJobHandler(jobRepo domain.JobRepository, deadRepo domain.DeadLetterRepository) *AdminJobHandler {
	return &AdminJobHandler{jobRepo: jobRepo, deadRepo: deadRepo}
}

// Submit 提交任务
func (h *AdminJobHandler) Submit(c *gin.Context) {
	var req struct {
		Type        string `json:"type" binding:"required"`
		Payload     string `json:"payload"`
		Priority    int    `json:"priority"`
		MaxAttempts int    `json:"max_attempts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	job := &domain.Job{
		Type:     req.Type,
		Payload:  req.Payload,
		Status:   domain.JobStatusPending,
		Priority: req.Priority,
		Attempts: 0,
	}
	if req.Priority == 0 {
		job.Priority = 5
	}
	if req.MaxAttempts > 0 {
		job.MaxAttempts = req.MaxAttempts
	} else {
		job.MaxAttempts = 3
	}
	if err := h.jobRepo.Create(c.Request.Context(), job); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"job_id": job.ID})
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

// Retry 手动重试：重置任务为 pending，清除错误
func (h *AdminJobHandler) Retry(c *gin.Context) {
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
	job.Status = domain.JobStatusPending
	job.Attempts = 0
	job.Error = ""
	if err := h.jobRepo.Update(c.Request.Context(), job); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"retried": id})
}

// ListDeadLetters 获取死信列表
func (h *AdminJobHandler) ListDeadLetters(c *gin.Context) {
	if h.deadRepo == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "dead letter repository not configured"))
		return
	}
	resolved := c.Query("resolved") == "true"
	letters, total, err := h.deadRepo.List(c.Request.Context(), 0, 20, resolved)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, letters, total, 1, 20)
}

// ResolveDeadLetter 处理死信
func (h *AdminJobHandler) ResolveDeadLetter(c *gin.Context) {
	if h.deadRepo == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "dead letter repository not configured"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.deadRepo.MarkResolved(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"resolved": id})
}
