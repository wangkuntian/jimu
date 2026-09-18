package domain

import (
	"context"
	"time"
)

// 登录结果
const (
	LoginStatusSuccess = "success"
	LoginStatusFailed  = "failed"
	LoginStatusLocked  = "locked"
)

// LoginHistory 登录历史：记录每次登录尝试（含失败与锁定），用于安全审计与异常登录排查。
type LoginHistory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"column:tenant_id;default:0" json:"tenant_id"`
	UserID    uint64    `gorm:"column:user_id;default:0;index" json:"user_id"`
	Username  string    `gorm:"size:64;not null" json:"username"`
	Status    string    `gorm:"size:16;not null" json:"status"`
	Reason    string    `gorm:"size:128;not null;default:''" json:"reason,omitempty"`
	IP        string    `gorm:"size:64;not null;default:''" json:"ip"`
	UserAgent string    `gorm:"size:256;not null;default:''" json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func (LoginHistory) TableName() string { return "login_histories" }

// LoginHistoryRepository 登录历史仓储接口
type LoginHistoryRepository interface {
	Create(ctx context.Context, record *LoginHistory) error
	// ListByUser 按用户查询登录历史（tenantID=0 表示平台级，不过滤）
	ListByUser(ctx context.Context, tenantID, userID uint64, offset, limit int) ([]LoginHistory, int64, error)
}
