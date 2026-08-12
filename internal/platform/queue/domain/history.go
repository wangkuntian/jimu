package domain

import (
	"context"
	"time"
)

// JobHistory 任务执行历史
type JobHistory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	JobID     uint64    `gorm:"not null;index" json:"job_id"`
	Status    string    `gorm:"size:16;not null" json:"status"`
	Error     string    `gorm:"type:text" json:"error"`
	Duration  int64     `json:"duration_ms"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

func (JobHistory) TableName() string { return "job_history" }

// JobHistoryRepository 历史仓储接口
type JobHistoryRepository interface {
	Create(ctx context.Context, h *JobHistory) error
	ListByJobID(ctx context.Context, jobID uint64) ([]JobHistory, error)
}
