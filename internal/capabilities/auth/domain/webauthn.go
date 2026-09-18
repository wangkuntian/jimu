package domain

import (
	"context"
	"time"
)

// WebAuthnCredential WebAuthn/通行密钥凭证。
// 只保存验签必需的公钥与计数器，明文私钥永远不离开认证器；
// 登录成功后需回写 SignCount/标志（库用于克隆检测与备份状态跟踪）。
type WebAuthnCredential struct {
	ID                uint64     `gorm:"primaryKey" json:"id"`
	TenantID          uint64     `gorm:"column:tenant_id;default:0;index" json:"tenant_id"`
	UserID            uint64     `gorm:"column:user_id;default:0;index" json:"user_id"`
	CredentialID      string     `gorm:"size:512;not null;uniqueIndex" json:"credential_id"`
	PublicKey         []byte     `gorm:"type:blob;not null" json:"-"`
	AttestationType   string     `gorm:"size:32;not null;default:''" json:"attestation_type,omitempty"`
	AttestationFormat string     `gorm:"size:32;not null;default:''" json:"attestation_format,omitempty"`
	AAGUID            string     `gorm:"size:36;not null;default:''" json:"aaguid,omitempty"`
	SignCount         uint32     `gorm:"not null;default:0" json:"sign_count"`
	Transports        string     `gorm:"size:255;not null;default:''" json:"transports,omitempty"`
	BackupEligible    bool       `gorm:"not null;default:false" json:"backup_eligible"`
	BackupState       bool       `gorm:"not null;default:false" json:"backup_state"`
	UserPresent       bool       `gorm:"not null;default:false" json:"-"`
	UserVerified      bool       `gorm:"not null;default:false" json:"user_verified"`
	Name              string     `gorm:"size:64;not null;default:''" json:"name,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (WebAuthnCredential) TableName() string { return "webauthn_credentials" }

// WebAuthnCredentialRepository WebAuthn 凭证仓储接口
type WebAuthnCredentialRepository interface {
	Create(ctx context.Context, credential *WebAuthnCredential) error
	// FindByCredentialID 按凭证 ID 查询（不存在返回 gorm.ErrRecordNotFound）
	FindByCredentialID(ctx context.Context, credentialID string) (*WebAuthnCredential, error)
	ListByUser(ctx context.Context, tenantID, userID uint64) ([]WebAuthnCredential, error)
	CountByUser(ctx context.Context, userID uint64) (int64, error)
	// Touch 登录成功后回写签名计数器与备份状态（克隆检测依赖计数器单调递增）
	Touch(ctx context.Context, id uint64, signCount uint32, backupState bool, usedAt time.Time) error
	// UpdateName 重命名凭证
	UpdateName(ctx context.Context, tenantID, userID, id uint64, name string) error
	// Delete 删除指定凭证（userID 用于归属校验，避免越权删除他人凭证）
	Delete(ctx context.Context, tenantID, userID, id uint64) error
}
