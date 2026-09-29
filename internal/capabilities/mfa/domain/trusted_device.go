package domain

import (
	"context"
	"time"
)

// TrustedDevice 可信设备：登录时携带设备令牌可跳过 TOTP，密码始终必需。
// 令牌仅存 SHA-256 哈希；改密或登出全部设备时吊销。
type TrustedDevice struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	TenantID   uint64     `gorm:"column:tenant_id;default:0" json:"tenant_id"`
	UserID     uint64     `gorm:"column:user_id;default:0;index" json:"user_id"`
	TokenHash  string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	Label      string     `gorm:"size:64;not null;default:''" json:"label,omitempty"`
	IP         string     `gorm:"size:64;not null;default:''" json:"ip"`
	UserAgent  string     `gorm:"size:256;not null;default:''" json:"user_agent"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (TrustedDevice) TableName() string { return "trusted_devices" }

// TrustedDeviceRepository 可信设备仓储接口
type TrustedDeviceRepository interface {
	Create(ctx context.Context, device *TrustedDevice) error
	// FindByTokenHash 按令牌哈希查询（不存在返回 gorm.ErrRecordNotFound）
	FindByTokenHash(ctx context.Context, tokenHash string) (*TrustedDevice, error)
	// Touch 更新最近使用时间
	Touch(ctx context.Context, id uint64, usedAt time.Time) error
	ListByUser(ctx context.Context, tenantID, userID uint64) ([]TrustedDevice, error)
	// Delete 删除指定设备（userID 用于归属校验，避免越权删除他人设备）
	Delete(ctx context.Context, tenantID, userID, id uint64) error
	DeleteAllByUser(ctx context.Context, userID uint64) error
	// DeleteExpired 清理已过期设备（返回删除条数）
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
