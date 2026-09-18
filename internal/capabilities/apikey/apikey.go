package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"jimu/internal/capabilities/apikey/domain"
	auth "jimu/internal/kernel/auth"

	"gorm.io/gorm"
)

// APIKey 为 kernel/auth 机制视图的别名：本能力负责验证与存储，
// context 注入/读取的中间件两侧共享同一机制类型（见 kernel/auth/apikey_context.go）。
type APIKey = auth.APIKey

// apiKeyPrefix API Key 前缀
const apiKeyPrefix = "jimu_"

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	// GetByKeyHash 通过 key 哈希查找 API Key
	GetByKeyHash(ctx context.Context, hash string) (*APIKey, error)
	// UpdateLastUsed 更新最后使用时间
	UpdateLastUsed(ctx context.Context, id uint64, t time.Time) error
}

// APIKeyVerifier API Key 验证器
type APIKeyVerifier struct {
	store APIKeyStore
}

// NewAPIKeyVerifier 创建 API Key 验证器
func NewAPIKeyVerifier(store APIKeyStore) *APIKeyVerifier {
	return &APIKeyVerifier{store: store}
}

// HashKey 计算 API Key 的 SHA-256 哈希（用于存储和查找）
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Verify 验证 API Key，返回对应的 APIKey 信息
func (v *APIKeyVerifier) Verify(ctx context.Context, providedKey string) (*APIKey, error) {
	if !strings.HasPrefix(providedKey, apiKeyPrefix) {
		return nil, errors.New("invalid key format")
	}
	if len(providedKey) < 16 {
		return nil, errors.New("key too short")
	}

	hash := HashKey(providedKey)
	key, err := v.store.GetByKeyHash(ctx, hash)
	if err != nil {
		return nil, errors.New("invalid key")
	}
	if !key.Enabled {
		return nil, errors.New("key is disabled")
	}
	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		return nil, errors.New("key is expired")
	}

	// 更新最后使用时间（异步，不影响主流程）
	_ = v.store.UpdateLastUsed(ctx, key.ID, time.Now())
	return key, nil
}

// dbAPIKeyStore 基于 api_keys 表的 API Key 存储（DB 持久化实现）
type dbAPIKeyStore struct {
	db *gorm.DB
}

// NewDBAPIKeyStore 创建 DB API Key 存储，复用 admin 模块 api_keys 表
func NewDBAPIKeyStore(db *gorm.DB) APIKeyStore {
	return &dbAPIKeyStore{db: db}
}

func (s *dbAPIKeyStore) GetByKeyHash(ctx context.Context, hash string) (*APIKey, error) {
	var row domain.APIKey
	err := s.db.WithContext(ctx).Where("key_hash = ?", hash).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("key not found")
		}
		return nil, err
	}
	return rowToAPIKey(&row), nil
}

func (s *dbAPIKeyStore) UpdateLastUsed(ctx context.Context, id uint64, t time.Time) error {
	return s.db.WithContext(ctx).Model(&domain.APIKey{}).
		Where("id = ?", id).
		Update("last_used", t).Error
}

// rowToAPIKey 将 api_keys 表实体转换为 auth.APIKey 机制视图
func rowToAPIKey(row *domain.APIKey) *APIKey {
	key := &APIKey{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		KeyPrefix: row.KeyPrefix,
		Enabled:   row.Enabled,
		ExpiresAt: row.ExpiresAt,
		LastUsed:  row.LastUsed,
	}
	if row.Scopes != "" {
		var scopes []string
		if err := json.Unmarshal([]byte(row.Scopes), &scopes); err == nil {
			key.Scopes = scopes
		}
	}
	return key
}
