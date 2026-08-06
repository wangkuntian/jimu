package application

import (
	"context"
	"time"
)

// TaskInfo 任务信息
type TaskInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	LastRun    time.Time `json:"last_run,omitempty"`
	LastStatus string    `json:"last_status"`
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
type AdminTaskService struct{}

// NewAdminTaskService 创建任务调度管理服务
func NewAdminTaskService() *AdminTaskService {
	return &AdminTaskService{}
}

// ListTasks 获取任务列表
func (s *AdminTaskService) ListTasks() []TaskInfo {
	// 返回预定义的任务列表（实际应从 Scheduler 获取）
	return []TaskInfo{
		{ID: "outbox_process", Name: "Process Outbox Events", Cron: "@every 10s", Enabled: true, LastStatus: "success"},
		{ID: "cleanup", Name: "Data Cleanup", Cron: "0 3 * * *", Enabled: true, LastStatus: "success"},
		{ID: "metrics_collect", Name: "Collect DB Metrics", Cron: "@every 15s", Enabled: true, LastStatus: "success"},
	}
}

// TriggerTask 手动触发任务
func (s *AdminTaskService) TriggerTask(ctx context.Context, taskID string) error {
	// TODO: integrate with actual scheduler
	return nil
}

// ToggleTask 暂停/恢复任务
func (s *AdminTaskService) ToggleTask(ctx context.Context, taskID string) error {
	// TODO: integrate with actual scheduler
	return nil
}

// GetHistory 获取任务执行历史
func (s *AdminTaskService) GetHistory(taskID string) []TaskExecution {
	return []TaskExecution{}
}
