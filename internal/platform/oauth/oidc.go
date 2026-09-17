// internal/platform/oauth/oidc.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"jimu/internal/platform/httpclient"
)

// defaultOIDCScopes OIDC 登录所需的最小 scope（未配置 scopes 时使用）
var defaultOIDCScopes = []string{"openid", "profile", "email"}

// OIDCConfig 通用 OIDC 提供商配置：只要 IdP 暴露
// {issuer_url}/.well-known/openid-configuration 即可接入（Keycloak/Okta/Auth0/Azure AD 等）。
type OIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IssuerURL    string   // 签发者地址，如 https://keycloak.example.com/realms/acme
	Scopes       []string // 可选，默认 openid profile email
}

// OIDCEndpoints discovery 文档中本项目用到的端点
type OIDCEndpoints struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// OIDCProvider 通用 OIDC 提供商实现（授权码模式，服务端持有 client_secret）
type OIDCProvider struct {
	name   string
	cfg    OIDCConfig
	client *httpclient.Client

	mu        sync.Mutex
	endpoints *OIDCEndpoints // discovery 结果，成功后缓存
}

// NewOIDCProvider 创建 OIDC 提供商；name 同时是回调路由中的 provider 名
func NewOIDCProvider(name string, cfg OIDCConfig, client *httpclient.Client) *OIDCProvider {
	return &OIDCProvider{name: name, cfg: cfg, client: client}
}

// Name 返回提供商名称
func (p *OIDCProvider) Name() string { return p.name }

// AuthURL 构造授权跳转 URL（首次调用会拉取并缓存 discovery 文档）
func (p *OIDCProvider) AuthURL(ctx context.Context, state string) (string, error) {
	endpoints, err := p.discovery(ctx)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(endpoints.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("oidc authorize url: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	q.Set("scope", strings.Join(p.scopes(), " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Exchange 用授权码换取用户信息：token 端点取 access_token，再调用 userinfo 端点取 subject/邮箱/名称。
// 使用 client_secret_post 认证（client_id/client_secret 放在表单中）。
func (p *OIDCProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	endpoints, err := p.discovery(ctx)
	if err != nil {
		return nil, err
	}
	if endpoints.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("oidc provider %s: discovery 缺少 userinfo_endpoint", p.name)
	}

	ctx, cancel := context.WithTimeout(ctx, oauthTimeout)
	defer cancel()

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURL},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoints.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oidc token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read oidc token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc token endpoint returned %d", resp.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("unmarshal oidc token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("oidc provider %s: token 响应缺少 access_token", p.name)
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoints.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc userinfo request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userResp, err := p.client.Do(ctx, userReq)
	if err != nil {
		return nil, fmt.Errorf("oidc userinfo: %w", err)
	}
	defer userResp.Body.Close()
	userBody, err := io.ReadAll(userResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read oidc userinfo: %w", err)
	}
	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc userinfo endpoint returned %d", userResp.StatusCode)
	}

	var info struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := json.Unmarshal(userBody, &info); err != nil {
		return nil, fmt.Errorf("unmarshal oidc userinfo: %w", err)
	}
	if info.Sub == "" {
		return nil, fmt.Errorf("oidc provider %s: userinfo 缺少 sub", p.name)
	}
	name := info.Name
	if name == "" {
		name = info.PreferredUsername
	}
	return &UserInfo{Subject: info.Sub, Email: info.Email, Name: name}, nil
}

func (p *OIDCProvider) scopes() []string {
	if len(p.cfg.Scopes) > 0 {
		return p.cfg.Scopes
	}
	return defaultOIDCScopes
}

// discovery 拉取并缓存 IdP 的 discovery 文档；失败不缓存，下次调用重试。
// 同时校验文档 issuer 与配置一致，避免 discovery 被指向其他签发者。
func (p *OIDCProvider) discovery(ctx context.Context) (*OIDCEndpoints, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.endpoints != nil {
		return p.endpoints, nil
	}

	discoveryURL := strings.TrimSuffix(p.cfg.IssuerURL, "/") + "/.well-known/openid-configuration"
	ctx, cancel := context.WithTimeout(ctx, oauthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc discovery returned %d", resp.StatusCode)
	}
	var endpoints OIDCEndpoints
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("unmarshal oidc discovery: %w", err)
	}
	if strings.TrimSuffix(endpoints.Issuer, "/") != strings.TrimSuffix(p.cfg.IssuerURL, "/") {
		return nil, fmt.Errorf("oidc discovery issuer %q does not match configured %q", endpoints.Issuer, p.cfg.IssuerURL)
	}
	if endpoints.AuthorizationEndpoint == "" || endpoints.TokenEndpoint == "" {
		return nil, fmt.Errorf("oidc provider %s: discovery 缺少 authorization/token 端点", p.name)
	}
	p.endpoints = &endpoints
	return p.endpoints, nil
}

var _ Provider = (*OIDCProvider)(nil)
