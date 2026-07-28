package domain

import (
	"context"
	"time"
)

// AuditLog 审计日志实体
type AuditLog struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	Username  string    `gorm:"size:64" json:"username"`
	Action    string    `gorm:"size:64" json:"action"`
	Resource  string    `gorm:"size:128" json:"resource"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	Method    string    `gorm:"size:16" json:"method"`
	Path      string    `gorm:"size:256" json:"path"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditRepository 审计日志仓储接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, offset, limit int) ([]AuditLog, int64, error)
}
