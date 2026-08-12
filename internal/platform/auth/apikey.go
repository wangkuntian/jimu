package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	adminapi "jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

// APIKey API 密钥信息
type APIKey struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	KeyPrefix string    `json:"key_prefix"` // 前 8 位，用于识别
	Scopes    []string  `json:"scopes"`     // 权限范围，如 ["read", "write"]
	Enabled   bool      `json:"enabled"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

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

// APIKeyContextKey context 中存储 API Key 的 key
type APIKeyContextKey struct{}

// ContextWithAPIKey 将 API Key 存入 context
func ContextWithAPIKey(ctx context.Context, key *APIKey) context.Context {
	return context.WithValue(ctx, APIKeyContextKey{}, key)
}

// APIKeyFromContext 从 context 获取已验证的 API Key
func APIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	val := ctx.Value(APIKeyContextKey{})
	if val == nil {
		return nil, false
	}
	key, ok := val.(*APIKey)
	return key, ok
}

// HasScope 检查 API Key 是否拥有指定 scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// ScopesString 将 scopes 序列化为存储格式
func ScopesString(scopes []string) string {
	return strings.Join(scopes, ",")
}

// ParseScopes 从存储格式解析 scopes
func ParseScopes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
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
	var row adminapi.APIKey
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
	return s.db.WithContext(ctx).Model(&adminapi.APIKey{}).
		Where("id = ?", id).
		Update("last_used", t).Error
}

// rowToAPIKey 将 admin 模块实体转换为 auth.APIKey
func rowToAPIKey(row *adminapi.APIKey) *APIKey {
	key := &APIKey{
		ID:        row.ID,
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

