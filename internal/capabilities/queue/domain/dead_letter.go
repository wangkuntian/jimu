package domain

import (
	"context"
	"time"
)

// DeadLetter 死信队列
type DeadLetter struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	TenantID   uint64    `gorm:"column:tenant_id;default:0;index" json:"tenant_id"` // 所属租户 ID（0=未归属）
	JobID      uint64    `gorm:"not null" json:"job_id"`
	Type       string    `gorm:"size:64;not null" json:"type"`
	Payload    string    `gorm:"type:text" json:"payload"`
	FailReason string    `gorm:"type:text" json:"fail_reason"`
	FailedAt   time.Time `json:"failed_at"`
	Resolved   bool      `gorm:"default:false;index" json:"resolved"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
}

func (DeadLetter) TableName() string { return "dead_letters" }

// DeadLetterRepository 死信仓储接口
type DeadLetterRepository interface {
	Create(ctx context.Context, d *DeadLetter) error
	List(ctx context.Context, tenantID uint64, offset, limit int, resolved bool) ([]DeadLetter, int64, error)
	// MarkResolved 标记死信已处理；租户不匹配时返回 gorm.ErrRecordNotFound
	MarkResolved(ctx context.Context, tenantID uint64, id uint64) error
}
