package domain

import (
	"context"
	"time"
)

// AuditLog 审计日志实体
type AuditLog struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	AdminID   uint64 `gorm:"not null;index" json:"admin_id"`
	AdminName string `gorm:"size:64" json:"admin_name"`
	Action    string `gorm:"size:64;not null;index" json:"action"`
	Resource  string `gorm:"size:128;not null" json:"resource"`
	Detail    string `gorm:"type:text" json:"detail"`
	IP        string `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName 返回表名
func (AuditLog) TableName() string { return "audit_logs" }

// AuditRepository 审计日志仓储接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]AuditLog, int64, error)
}
