package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// APIKey API 密钥实体
type APIKey struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	KeyPrefix string    `gorm:"size:16;not null;index" json:"key_prefix"`
	KeyHash   string    `gorm:"size:64;not null;index:idx_key_hash" json:"-"`
	Scopes    string    `gorm:"type:text" json:"-"` // JSON array
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	LastUsed  time.Time `json:"last_used,omitempty"`
	UseCount  int64     `gorm:"default:0" json:"use_count"`
	CreatedBy uint64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 返回表名
func (APIKey) TableName() string { return "api_keys" }

// APIKeyRepository API Key 仓储接口
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	FindByID(ctx context.Context, id uint64) (*APIKey, error)
	FindByKeyHash(ctx context.Context, hash string) (*APIKey, error)
	List(ctx context.Context, offset, limit int) ([]APIKey, int64, error)
	Update(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, id uint64) error
	IncrementUseCount(ctx context.Context, id uint64) error
}

// HashKey 计算 API Key 的 SHA-256 哈希
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
