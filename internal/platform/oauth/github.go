// internal/platform/oauth/github.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"jimu/internal/platform/httpclient"

	"golang.org/x/oauth2"
)

// GitHubConfig GitHub OAuth 配置
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GitHubProvider GitHub OAuth 实现
type GitHubProvider struct {
	config      *oauth2.Config
	client      *httpclient.Client
	userInfoURL string // 未导出，测试可覆盖
}

// NewGitHubProvider 创建 GitHub Provider
func NewGitHubProvider(cfg GitHubConfig, client *httpclient.Client) *GitHubProvider {
	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
		},
		client:      client,
		userInfoURL: "https://api.github.com/user",
	}
}

// Name 返回提供商名称
func (p *GitHubProvider) Name() string { return "github" }

// AuthURL 构造授权跳转 URL
func (p *GitHubProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange 用授权码换取用户信息
func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthTimeout)
	defer cancel()
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github exchange: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("github userinfo req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := p.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("github userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read github userinfo: %w", err)
	}
	var info struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("unmarshal github userinfo: %w", err)
	}
	return &UserInfo{
		Subject: fmt.Sprintf("%d", info.ID),
		Email:   info.Email,
		Name:    info.Login,
	}, nil
}

var _ Provider = (*GitHubProvider)(nil)
