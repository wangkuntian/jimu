package domain

import (
	"context"
	"time"
)

// JobStatus 任务状态
type JobStatus = string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusSuccess JobStatus = "success"
	JobStatusFailed  JobStatus = "failed"
	JobStatusDead    JobStatus = "dead"
)

// Job 任务实体
type Job struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Type        string    `gorm:"size:64;not null;index" json:"type"`
	Payload     string    `gorm:"type:text" json:"payload"`
	Status      string    `gorm:"size:16;not null;default:pending;index:idx_status_next_run" json:"status"`
	Priority    int       `gorm:"default:5" json:"priority"`
	Attempts    int       `gorm:"default:0" json:"attempts"`
	MaxAttempts int       `gorm:"default:3" json:"max_attempts"`
	NextRunAt   time.Time `gorm:"index:idx_status_next_run" json:"next_run_at"`
	Error       string    `gorm:"type:text" json:"error"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Job) TableName() string { return "jobs" }

// JobRepository 任务仓储接口
type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	FindByID(ctx context.Context, id uint64) (*Job, error)
	Update(ctx context.Context, job *Job) error
	List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]Job, int64, error)
}
