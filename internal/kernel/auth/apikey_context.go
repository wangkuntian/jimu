package auth

import (
	"context"
	"time"
)

// APIKey 已认证 API Key 的机制视图（纯身份信息，不含 gorm 标签；
// api_keys 表实体归 apikey 能力的 domain 包）。
type APIKey struct {
	ID        uint64    `json:"id"`
	TenantID  uint64    `json:"tenant_id"` // 所属租户（Key 决定租户，客户端不可指定）
	Name      string    `json:"name"`
	KeyPrefix string    `json:"key_prefix"` // 前 8 位，用于识别
	Scopes    []string  `json:"scopes"`     // 权限范围，如 ["read", "write"]
	Enabled   bool      `json:"enabled"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

// HasScope 检查 API Key 是否拥有指定 scope。
// 空 scopes 表示拒绝一切；只有显式包含 "*" 才代表全权。
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
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
