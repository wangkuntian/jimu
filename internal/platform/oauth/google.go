// internal/platform/oauth/google.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"jimu/internal/platform/httpclient"

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
	config      *oauth2.Config
	client      *httpclient.Client
	userInfoURL string // 未导出，测试可覆盖
}

// NewGoogleProvider 创建 Google Provider
func NewGoogleProvider(cfg GoogleConfig, client *httpclient.Client) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		client:      client,
		userInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
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
	ctx, cancel := context.WithTimeout(ctx, oauthTimeout)
	defer cancel()
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("google userinfo req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := p.client.Do(ctx, req)
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
