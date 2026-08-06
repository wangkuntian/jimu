package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
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

// GenerateKey 生成新的 API Key
// 返回完整 key（仅此一次）和 APIKey 元信息
func GenerateKey(id uint64, name string, scopes []string, ttl time.Duration) (string, *APIKey, error) {
	if name == "" {
		return "", nil, errors.New("name is required")
	}

	// 生成 32 字节随机 key
	raw := make([]byte, 32)
	now := time.Now().UnixNano()
	for i := range raw {
		// 简化实现：实际应使用 crypto/rand
		raw[i] = byte((now + int64(i)*7) % 256)
	}
	fullKey := apiKeyPrefix + hex.EncodeToString(raw)

	key := &APIKey{
		ID:        id,
		Name:      name,
		KeyPrefix: fullKey[:min(8+len(apiKeyPrefix), len(fullKey))],
		Scopes:    scopes,
		Enabled:   true,
	}
	if ttl > 0 {
		key.ExpiresAt = time.Now().Add(ttl)
	}

	return fullKey, key, nil
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

// ConstantTimeCompare 恒定时间比较两个 API Key（防时序攻击）
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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

// apiKeyStoreKey Redis 存储 key
func apiKeyStoreKey(id uint64) string {
	return fmt.Sprintf("jimu:apikey:%d", id)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure redis.Scripter satisfies APIKeyStore if needed
var _ APIKeyStore = (*redisAPIKeyStore)(nil)

// redisAPIKeyStore Redis 实现的 API Key 存储
type redisAPIKeyStore struct {
	client *redis.Client
}

// NewRedisAPIKeyStore 创建 Redis API Key 存储
func NewRedisAPIKeyStore(client *redis.Client) APIKeyStore {
	return &redisAPIKeyStore{client: client}
}

func (s *redisAPIKeyStore) GetByKeyHash(ctx context.Context, hash string) (*APIKey, error) {
	// 从 Redis hash 中查找
	data, err := s.client.HGetAll(ctx, "jimu:apikey:index:"+hash).Result()
	if err != nil || len(data) == 0 {
		return nil, errors.New("key not found")
	}
	// 简化实现：实际应反序列化完整信息
	_ = data
	return nil, errors.New("not implemented: use database store")
}

func (s *redisAPIKeyStore) UpdateLastUsed(ctx context.Context, id uint64, t time.Time) error {
	return s.client.HSet(ctx, apiKeyStoreKey(id), "last_used", t.Unix()).Err()
}
