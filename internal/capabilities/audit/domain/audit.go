package domain

import (
	"context"
	"time"
)

// Change 记录单个字段的变更
type Change struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// AuditLog 审计日志实体
type AuditLog struct {
	ID         uint64   `gorm:"primaryKey" json:"id"`
	UserID     uint64   `json:"user_id"`
	TenantID   uint64   `gorm:"column:tenant_id;default:0" json:"tenant_id"` // 所属租户 ID（0=未归属）
	Username   string   `gorm:"size:64" json:"username"`
	Action     string   `gorm:"size:64" json:"action"`
	Resource   string   `gorm:"size:128" json:"resource"`
	Detail     string   `gorm:"type:text" json:"detail"`
	ChangesRaw string   `gorm:"column:changes;type:text" json:"-"`
	Changes    []Change `gorm:"-" json:"changes,omitempty"`
	IP         string   `gorm:"size:64" json:"ip"`
	Method     string   `gorm:"size:16" json:"method"`
	Path       string   `gorm:"size:256" json:"path"`
	Status     int      `json:"status"`
	// 链式哈希：prev_hash 为上一条的 entry_hash，entry_hash 由 domain.ComputeEntryHash 计算
	PrevHash  string    `gorm:"size:64;not null;default:''" json:"prev_hash,omitempty"`
	EntryHash string    `gorm:"size:64;not null;default:''" json:"entry_hash,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditRepository 审计日志仓储接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	CreateBatch(ctx context.Context, logs []AuditLog) error
	FindByID(ctx context.Context, id uint64) (*AuditLog, error)
	List(ctx context.Context, tenantID uint64, offset, limit int, sort, order string) ([]AuditLog, int64, error)
	// ListForVerify 按 id 升序返回待校验条目；tenantID=0 表示平台级（不过滤）
	ListForVerify(ctx context.Context, tenantID uint64, fromID, toID uint64, limit int) ([]AuditLog, error)
	// ChainHead 返回该租户审计链最新哈希；无记录时返回空串
	ChainHead(ctx context.Context, tenantID uint64) (string, error)
	// CountRange 统计时间范围内（created_at >= start 且 < end）的条目数，用于导出前限流校验
	CountRange(ctx context.Context, tenantID uint64, start, end time.Time) (int64, error)
	// ListRange 按时间范围分页返回条目（id 升序，便于稳定分批导出）
	ListRange(ctx context.Context, tenantID uint64, start, end time.Time, offset, limit int) ([]AuditLog, error)
}
