// internal/platform/oauth/provider.go
package oauth

import "context"

// UserInfo OAuth 用户信息
type UserInfo struct {
	Subject string // provider 内唯一 ID
	Email   string
	Name    string
}

// Provider 第三方登录提供商接口
type Provider interface {
	// Name 提供商名称（google/github/wechat）
	Name() string
	// AuthURL 构造授权跳转 URL
	AuthURL(state string) string
	// Exchange 用授权码换取用户信息
	Exchange(ctx context.Context, code string) (*UserInfo, error)
}
