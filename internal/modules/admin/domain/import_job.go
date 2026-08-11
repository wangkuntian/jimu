package domain

import (
	"context"
	"time"
)

// ImportJobStatus 导入任务状态
type ImportJobStatus = string

const (
	ImportJobPending    ImportJobStatus = "pending"
	ImportJobProcessing ImportJobStatus = "processing"
	ImportJobCompleted  ImportJobStatus = "completed"
	ImportJobFailed     ImportJobStatus = "failed"
)

// ImportJob 数据导入任务
type ImportJob struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Type        string    `gorm:"size:64;not null" json:"type"`
	Filename    string    `gorm:"size:255;not null" json:"filename"`
	Status      string    `gorm:"size:16;not null;default:pending;index" json:"status"`
	TotalRows   int       `gorm:"not null;default:0" json:"total_rows"`
	SuccessRows int       `gorm:"not null;default:0" json:"success_rows"`
	ErrorRows   int       `gorm:"not null;default:0" json:"error_rows"`
	Errors      string    `gorm:"type:text" json:"errors,omitempty"` // JSON 错误详情
	CreatedBy   uint64    `gorm:"index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

func (ImportJob) TableName() string { return "import_jobs" }

// ImportJobRepository 导入任务仓储接口
type ImportJobRepository interface {
	Create(ctx context.Context, job *ImportJob) error
	FindByID(ctx context.Context, id uint64) (*ImportJob, error)
	Update(ctx context.Context, job *ImportJob) error
}
