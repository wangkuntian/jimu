package domain

import (
	"context"
	"time"
)

// UserMFA 用户 TOTP 二次验证状态。密钥 AES-GCM 落库（encryption:"true"），
// 读取时由全局加密 hook 解密回明文。
type UserMFA struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"column:tenant_id;default:0;index" json:"tenant_id"`
	UserID      uint64    `gorm:"column:user_id;default:0;index" json:"user_id"`
	TOTPSecret  string    `gorm:"column:totp_secret;type:text" encryption:"true" json:"-"`
	TOTPEnabled bool      `gorm:"column:totp_enabled;default:false" json:"totp_enabled"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

func (UserMFA) TableName() string { return "user_mfa" }

// MFARepository TOTP 状态仓储接口。
type MFARepository interface {
	// FindByUser 查用户 TOTP 状态；不存在返回 gorm.ErrRecordNotFound。
	FindByUser(ctx context.Context, userID uint64) (*UserMFA, error)
	// UpsertSecret 写入密钥并设置启用状态，存在则更新（设置密钥 + enabled）。
	UpsertSecret(ctx context.Context, tenantID, userID uint64, secret string, enabled bool) error
	// Clear 清除该用户的 TOTP 记录（关闭二次验证）。
	Clear(ctx context.Context, userID uint64) error
}
