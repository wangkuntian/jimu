package application

import (
	"context"
	"time"

	"jimu/internal/platform/scheduler"
	apperrors "jimu/internal/shared/errors"
)

// TaskInfo 任务信息
type TaskInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	LastRun    time.Time `json:"last_run,omitempty"`
	LastStatus string    `json:"last_status"`
	LastError  string    `json:"last_error,omitempty"`
	RunCount   int64     `json:"run_count"`
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"` // success / failed
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Error     string    `json:"error,omitempty"`
}

// AdminTaskService 任务调度管理服务
type AdminTaskService struct {
	sched *scheduler.CronScheduler
}

// NewAdminTaskService 创建任务调度管理服务
func NewAdminTaskService(sched *scheduler.CronScheduler) *AdminTaskService {
	return &AdminTaskService{sched: sched}
}

// ListTasks 获取任务列表
func (s *AdminTaskService) ListTasks() []TaskInfo {
	if s.sched == nil {
		return []TaskInfo{}
	}
	jobs := s.sched.Jobs()
	result := make([]TaskInfo, 0, len(jobs))
	for _, j := range jobs {
		result = append(result, TaskInfo{
			ID:         j.ID,
			Name:       j.Name,
			Cron:       j.Cron,
			Enabled:    j.Enabled,
			LastRun:    j.LastRun,
			LastStatus: j.LastStatus,
			LastError:  j.LastError,
			RunCount:   j.RunCount,
		})
	}
	return result
}

// TriggerTask 手动触发任务
func (s *AdminTaskService) TriggerTask(ctx context.Context, taskID string) error {
	if s.sched == nil {
		return apperrors.New(apperrors.CodeInternalError, "scheduler not configured")
	}
	return s.sched.TriggerJob(ctx, taskID)
}

// ToggleTask 暂停/恢复任务
func (s *AdminTaskService) ToggleTask(ctx context.Context, taskID string) error {
	if s.sched == nil {
		return apperrors.New(apperrors.CodeInternalError, "scheduler not configured")
	}
	// 根据当前状态切换
	jobs := s.sched.Jobs()
	for _, j := range jobs {
		if j.ID == taskID {
			return s.sched.SetEnabled(ctx, taskID, !j.Enabled)
		}
	}
	return apperrors.New(apperrors.CodeNotFound, "task not found: "+taskID)
}

// GetHistory 获取任务执行历史
// 说明：调度器当前仅在内存记录最近一次运行状态（JobInfo），无持久化执行历史
func (s *AdminTaskService) GetHistory(taskID string) []TaskExecution {
	if s.sched == nil {
		return []TaskExecution{}
	}
	for _, j := range s.sched.Jobs() {
		if j.ID == taskID {
			exec := TaskExecution{
				TaskID: j.ID,
				Status: j.LastStatus,
				Error:  j.LastError,
			}
			if !j.LastRun.IsZero() {
				exec.StartedAt = j.LastRun
			}
			return []TaskExecution{exec}
		}
	}
	return []TaskExecution{}
}
