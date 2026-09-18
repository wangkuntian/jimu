package domain

import (
	"context"
	"time"
)

// PasswordHistory 密码历史：保存用户历史密码哈希，用于阻止改回旧密码。
type PasswordHistory struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"column:tenant_id;default:0" json:"tenant_id"`
	UserID       uint64    `gorm:"column:user_id;default:0;index" json:"user_id"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (PasswordHistory) TableName() string { return "password_histories" }

// PasswordHistoryRepository 密码历史仓储接口
type PasswordHistoryRepository interface {
	// Add 记录一条历史密码哈希
	Add(ctx context.Context, tenantID, userID uint64, passwordHash string) error
	// ListRecentHashes 返回该用户最近 limit 条历史密码哈希（新→旧）
	ListRecentHashes(ctx context.Context, userID uint64, limit int) ([]string, error)
	// Trim 仅保留最近 keep 条，删除更早的记录（防止无限增长）
	Trim(ctx context.Context, userID uint64, keep int) error
}
