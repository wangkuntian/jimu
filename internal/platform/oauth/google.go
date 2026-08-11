// internal/platform/oauth/google.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleConfig Google OAuth 配置
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GoogleProvider Google OAuth 实现
type GoogleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider 创建 Google Provider
func NewGoogleProvider(cfg GoogleConfig) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// Name 返回提供商名称
func (p *GoogleProvider) Name() string { return "google" }

// AuthURL 构造授权跳转 URL
func (p *GoogleProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange 用授权码换取用户信息
func (p *GoogleProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google userinfo: %w", err)
	}
	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("unmarshal google userinfo: %w", err)
	}
	return &UserInfo{Subject: info.ID, Email: info.Email, Name: info.Name}, nil
}

var _ Provider = (*GoogleProvider)(nil)
