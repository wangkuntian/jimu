# OAuth2/OIDC Third-Party Login Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 OAuth2/OIDC 第三方登录模块，支持 Google/GitHub/微信，用户经绑定表关联，登录后签发 JWT 复用现有 auth 体系。

**Architecture:** `internal/platform/oauth/` 放 Provider 抽象与实现（Google/GitHub/WeChat），`internal/modules/oauth/` 放业务模块（handler + service）。OAuth 回调经 `Exchange` 拿 UserInfo，查 `user_oauth_bindings` 表匹配或创建用户 + 绑定记录，再用 `auth.JWT` + `auth.SessionStore` 签发 token 创建 session。新增迁移 `013_create_user_oauth_bindings.sql`。

**Tech Stack:** Go 1.26.5, `golang.org/x/oauth2`, 现有 `auth.JWT`/`auth.SessionStore`/`userdomain.UserRepository`

## Global Constraints

- 模块：`jimu`
- 新依赖：`golang.org/x/oauth2`
- 迁移序号：`013`（现有最大 `012`）
- OAuth 用户绑定表 `user_oauth_bindings`：一对多（一个用户可绑多个 provider）
- 回调创建用户时，用户名生成规则：`{provider}_{subject}` 前 64 字符（`users.username` 限长 64）
- 新错误码段：`3xxx` OAuth 模块
- 遵循现有代码风格：中文注释、swagger 中文注解、统一响应

---

### Task 1: Provider 抽象 + Google/GitHub/WeChat 实现

**Files:**
- Create: `internal/platform/oauth/provider.go`
- Create: `internal/platform/oauth/google.go`
- Create: `internal/platform/oauth/github.go`
- Create: `internal/platform/oauth/wechat.go`
- Create: `internal/platform/oauth/provider_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `golang.org/x/oauth2`
- Produces:
  - `UserInfo` struct：`Subject string`（provider 内唯一 ID）、`Email string`、`Name string`
  - `Provider` 接口：`Name() string`、`AuthURL(state string) string`、`Exchange(ctx context.Context, code string) (*UserInfo, error)`
  - `NewGoogleProvider(cfg GoogleConfig) *GoogleProvider`、`NewGitHubProvider(cfg GitHubConfig) *GitHubProvider`、`NewWeChatProvider(cfg WeChatConfig) *WeChatProvider`

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/oauth/provider_test.go
package oauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvidersImplementInterface(t *testing.T) {
	var _ Provider = (*GoogleProvider)(nil)
	var _ Provider = (*GitHubProvider)(nil)
	var _ Provider = (*WeChatProvider)(nil)
}

func TestGoogleAuthURL(t *testing.T) {
	p := NewGoogleProvider(GoogleConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/api/v1/oauth/google/callback",
	})
	url := p.AuthURL("state123")
	assert.Contains(t, url, "state=state123")
	assert.Contains(t, url, "client_id=id")
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/oauth/ -v`
Expected: 编译失败，包/类型未定义

- [ ] **Step 3: 引入依赖**

Run: `go get golang.org/x/oauth2@latest`

- [ ] **Step 4: 写 `provider.go`**

```go
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
```

- [ ] **Step 5: 写 `google.go`**

```go
// internal/platform/oauth/google.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
```

- [ ] **Step 6: 写 `github.go`**

```go
// internal/platform/oauth/github.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	config *oauth2.Config
}

// NewGitHubProvider 创建 GitHub Provider
func NewGitHubProvider(cfg GitHubConfig) *GitHubProvider {
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
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github exchange: %w", err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
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
```

- [ ] **Step 7: 写 `wechat.go`**

```go
// internal/platform/oauth/wechat.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// WeChatConfig 微信 OAuth 配置
type WeChatConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// WeChatProvider 微信 OAuth 实现
type WeChatProvider struct {
	config *oauth2.Config
}

// NewWeChatProvider 创建微信 Provider
func NewWeChatProvider(cfg WeChatConfig) *WeChatProvider {
	return &WeChatProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://open.weixin.qq.com/connect/qrconnect",
				TokenURL: "https://api.weixin.qq.com/sns/oauth2/access_token",
			},
		},
	}
}

// Name 返回提供商名称
func (p *WeChatProvider) Name() string { return "wechat" }

// AuthURL 构造授权跳转 URL
func (p *WeChatProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange 用授权码换取用户信息
func (p *WeChatProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("wechat exchange: %w", err)
	}
	resp, err := http.Get("https://api.weixin.qq.com/sns/userinfo?access_token=" + token.AccessToken + "&openid=" + token.Extra("openid").(string))
	if err != nil {
		return nil, fmt.Errorf("wechat userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat userinfo: %w", err)
	}
	var info struct {
		OpenID string `json:"openid"`
		UnionID string `json:"unionid"`
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("unmarshal wechat userinfo: %w", err)
	}
	return &UserInfo{Subject: info.OpenID, Email: "", Name: info.Nickname}, nil
}

var _ Provider = (*WeChatProvider)(nil)
```

- [ ] **Step 8: 运行测试验证通过**

Run: `go test ./internal/platform/oauth/ -v`
Expected: PASS

- [ ] **Step 9: 提交**

```bash
git add internal/platform/oauth/ go.mod go.sum
git commit -m "feat(oauth): add provider abstraction with google/github/wechat"
```

---

### Task 2: 绑定表迁移 + OAuth 模块骨架

**Files:**
- Create: `migrations/013_create_user_oauth_bindings.sql`
- Create: `internal/modules/oauth/module.go`
- Create: `internal/modules/oauth/domain/binding.go`
- Create: `internal/modules/oauth/domain/repository.go`
- Create: `internal/modules/oauth/infrastructure/mysql_repository.go`
- Create: `internal/modules/oauth/interfaces/router.go`
- Create: `internal/modules/oauth/interfaces/handler.go`
- Create: `internal/modules/oauth/application/service.go`
- Create: `internal/modules/oauth/application/dto.go`

**Interfaces:**
- Consumes: `Provider` 接口（Task 1）、`userdomain.UserRepository`
- Produces:
  - `OAuthBinding` 实体：`ID`、`UserID uint64`、`Provider string`、`Subject string`、`CreatedAt`
  - `BindingRepository` 接口：`FindByProviderSubject(ctx, provider, subject) (*OAuthBinding, error)`、`Create(ctx, binding *OAuthBinding) error`
  - `OAuthService`：`NewOAuthService(userRepo, bindingRepo, jwtUtil, sessions, providers, accessMin)`，方法 `Login(ctx, providerName, code) (*TokenPair, error)`、`AuthURL(providerName, state) (string, error)`
  - `RegisterOAuthRoutes(r *gin.RouterGroup, service)`：`GET /oauth/{provider}/login`、`GET /oauth/{provider}/callback`

- [ ] **Step 1: 写迁移**

```sql
-- migrations/013_create_user_oauth_bindings.sql
-- +goose Up
CREATE TABLE user_oauth_bindings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户 ID',
    provider VARCHAR(32) NOT NULL COMMENT '提供商（google/github/wechat）',
    subject VARCHAR(128) NOT NULL COMMENT '提供商内唯一 ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_oauth_binding (provider, subject),
    KEY idx_oauth_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录绑定表';

-- +goose Down
DROP TABLE user_oauth_bindings;
```

- [ ] **Step 2: 写 domain 实体**

```go
// internal/modules/oauth/domain/binding.go
package domain

import "time"

// OAuthBinding 第三方登录绑定记录
type OAuthBinding struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"not null" json:"user_id"`
	Provider  string    `gorm:"size:32;not null" json:"provider"`
	Subject   string    `gorm:"size:128;not null" json:"subject"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (OAuthBinding) TableName() string { return "user_oauth_bindings" }
```

```go
// internal/modules/oauth/domain/repository.go
package domain

import "context"

// BindingRepository 绑定记录仓储接口
type BindingRepository interface {
	// FindByProviderSubject 按 provider+subject 查绑定
	FindByProviderSubject(ctx context.Context, provider, subject string) (*OAuthBinding, error)
	// Create 创建绑定
	Create(ctx context.Context, binding *OAuthBinding) error
}
```

```go
// internal/modules/oauth/infrastructure/mysql_repository.go
package infrastructure

import (
	"context"

	"jimu/internal/modules/oauth/domain"

	"gorm.io/gorm"
)

// MySQLBindingRepository 基于 MySQL 的绑定仓储
type MySQLBindingRepository struct {
	db *gorm.DB
}

// NewMySQLBindingRepository 创建绑定仓储
func NewMySQLBindingRepository(db *gorm.DB) *MySQLBindingRepository {
	return &MySQLBindingRepository{db: db}
}

// FindByProviderSubject 按 provider+subject 查绑定
func (r *MySQLBindingRepository) FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.OAuthBinding, error) {
	var binding domain.OAuthBinding
	err := r.db.WithContext(ctx).
		Where("provider = ? AND subject = ?", provider, subject).
		First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// Create 创建绑定
func (r *MySQLBindingRepository) Create(ctx context.Context, binding *domain.OAuthBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

var _ domain.BindingRepository = (*MySQLBindingRepository)(nil)
```

- [ ] **Step 3: 写 application service**

```go
// internal/modules/oauth/application/service.go
package application

import (
	"context"
	"fmt"

	oauthdomain "jimu/internal/modules/oauth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/modules/auth/domain"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"
	"jimu/internal/shared/errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// OAuthService OAuth 登录服务
type OAuthService struct {
	userRepo   userdomain.UserRepository
	bindingRepo oauthdomain.BindingRepository
	jwtUtil    *auth.JWT
	sessions   auth.SessionStore
	providers  map[string]oauthplatform.Provider
	accessMin  int
}

// NewOAuthService 创建 OAuth 服务
func NewOAuthService(userRepo userdomain.UserRepository, bindingRepo oauthdomain.BindingRepository, jwtUtil *auth.JWT, sessions auth.SessionStore, providers map[string]oauthplatform.Provider, accessMin int) *OAuthService {
	return &OAuthService{
		userRepo:    userRepo,
		bindingRepo: bindingRepo,
		jwtUtil:     jwtUtil,
		sessions:    sessions,
		providers:   providers,
		accessMin:   accessMin,
	}
}

// AuthURL 构造授权跳转 URL
func (s *OAuthService) AuthURL(providerName, state string) (string, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return "", errors.New(errors.CodeInternalError, "unknown oauth provider: "+providerName)
	}
	return p.AuthURL(state), nil
}

// Login 处理 OAuth 回调，匹配/创建用户并签发 token
func (s *OAuthService) Login(ctx context.Context, providerName, code string) (*domain.TokenPair, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return nil, errors.New(errors.CodeInternalError, "unknown oauth provider: "+providerName)
	}
	info, err := p.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "oauth exchange failed", err)
	}

	// 匹配绑定
	binding, err := s.bindingRepo.FindByProviderSubject(ctx, providerName, info.Subject)
	var userID uint64
	if err == nil {
		userID = binding.UserID
	} else {
		// 创建用户 + 绑定
		username := fmt.Sprintf("%s_%s", providerName, info.Subject)
		if len(username) > 64 {
			username = username[:64]
		}
		randPwd := uuid.NewString()[:16]
		hashed, err := bcrypt.GenerateFromPassword([]byte(randPwd), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "hash oauth password", err)
		}
		user := &userdomain.User{
			Username: username,
			Password: string(hashed),
			Status:   1,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "create oauth user", err)
		}
		userID = user.ID
		if err := s.bindingRepo.Create(ctx, &oauthdomain.OAuthBinding{
			UserID:   userID,
			Provider: providerName,
			Subject:  info.Subject,
		}); err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "create oauth binding", err)
		}
	}

	// 签发 token
	sessionID := uuid.NewString()
	accessToken, err := s.jwtUtil.GenerateAccess(userID, sessionID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "generate access token", err)
	}
	refreshToken, refreshClaims, err := s.jwtUtil.GenerateRefresh(userID, sessionID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "generate refresh token", err)
	}
	if err := s.sessions.Create(ctx, userID, sessionID, refreshClaims.ID, refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time)); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "create session", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}
```

- [ ] **Step 4: 写 DTO + handler + router**

```go
// internal/modules/oauth/application/dto.go
package application

// LoginRequest 回调登录请求
type LoginRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
```

```go
// internal/modules/oauth/interfaces/handler.go
package interfaces

import (
	"jimu/internal/modules/oauth/application"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// OAuthHandler OAuth HTTP 处理器
type OAuthHandler struct {
	service *application.OAuthService
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler(service *application.OAuthService) *OAuthHandler {
	return &OAuthHandler{service: service}
}

// Login godoc
// @Summary      OAuth 登录跳转
// @Description  重定向到第三方授权页
// @Tags         OAuth
// @Param        provider path string true "提供商 (google/github/wechat)"
// @Param        state   query string true "防 CSRF 状态"
// @Success      302
// @Router       /oauth/{provider}/login [get]
func (h *OAuthHandler) Login(c *gin.Context) {
	providerName := c.Param("provider")
	url, err := h.service.AuthURL(providerName, c.Query("state"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Redirect(302, url)
}

// Callback godoc
// @Summary      OAuth 回调
// @Description  处理第三方授权回调，签发 JWT
// @Tags         OAuth
// @Param        provider path string true "提供商"
// @Param        code      query string true "授权码"
// @Param        state     query string true "状态"
// @Success      200       {object} response.Body "登录成功"
// @Router       /oauth/{provider}/callback [get]
func (h *OAuthHandler) Callback(c *gin.Context) {
	providerName := c.Param("provider")
	tokenPair, err := h.service.Login(c.Request.Context(), providerName, c.Query("code"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}
```

```go
// internal/modules/oauth/interfaces/router.go
package interfaces

import (
	"jimu/internal/modules/oauth/application"

	"github.com/gin-gonic/gin"
)

// RegisterOAuthRoutes 注册 OAuth 路由
func RegisterOAuthRoutes(r *gin.RouterGroup, service *application.OAuthService) {
	handler := NewOAuthHandler(service)
	group := r.Group("/oauth")
	{
		group.GET("/:provider/login", handler.Login)
		group.GET("/:provider/callback", handler.Callback)
	}
}
```

- [ ] **Step 5: 写 module.go**

```go
// internal/modules/oauth/module.go
package oauth

import (
	"jimu/internal/contract"
	oauthapp "jimu/internal/modules/oauth/application"
	"jimu/internal/modules/oauth/interfaces"

	"github.com/gin-gonic/gin"
)

// Module OAuth 模块
type Module struct {
	service *oauthapp.OAuthService
}

// New 创建 OAuth 模块
func New(service *oauthapp.OAuthService) *Module {
	return &Module{service: service}
}

// Name 模块名
func (m *Module) Name() string { return "oauth" }

// RegisterHTTP 注册 HTTP 路由
func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterOAuthRoutes(rg, m.service)
}

// RegisterJobs 无定时任务
func (m *Module) RegisterJobs(j contract.JobRegistry) {}

// RegisterEvents 无事件
func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
```

- [ ] **Step 6: 运行迁移测试**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 7: 提交**

```bash
git add migrations/013_create_user_oauth_bindings.sql internal/modules/oauth/
git commit -m "feat(oauth): add binding table and module skeleton"
```

---

### Task 3: 装配 OAuth 模块 + 配置

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/validate.go`
- Modify: `internal/app/container.go`
- Modify: `internal/app/bootstrap.go`
- Modify: `cmd/server/main.go`
- Modify: `configs/app.yaml`
- Modify: `configs/app.prod.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: `NewOAuthService`（Task 2）、`NewGoogleProvider`/`NewGitHubProvider`/`NewWeChatProvider`（Task 1）
- Produces: `container.go` 构建 OAuth 模块并注册；config 加 `oauth.providers` 配置

- [ ] **Step 1: 加配置结构**

在 `internal/config/config.go`：

```go
// OAuthProviderConfig 单个提供商配置
type OAuthProviderConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	Enabled      bool   `mapstructure:"enabled"`
}

// OAuthConfig OAuth 配置
type OAuthConfig struct {
	Providers map[string]OAuthProviderConfig `mapstructure:"providers"`
}
```

在 `Config` struct 加：`OAuth OAuthConfig \`mapstructure:"oauth"\``

- [ ] **Step 2: 加错误码**

在 `internal/shared/errors/errors.go` 加 OAuth 模块错误码段 `3xxx`（跟随现有模式）。

- [ ] **Step 3: 修改 container.go**

在 `Container` struct 加 `OAuthModule *oauth.Module`，并在 NewContainer 构建：

```go
	// OAuth
	var providers = make(map[string]oauthplatform.Provider)
	for name, pc := range cfg.OAuth.Providers {
		if !pc.Enabled {
			continue
		}
		switch name {
		case "google":
			providers[name] = oauthplatform.NewGoogleProvider(oauthplatform.GoogleConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			})
		case "github":
			providers[name] = oauthplatform.NewGitHubProvider(oauthplatform.GitHubConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			})
		case "wechat":
			providers[name] = oauthplatform.NewWeChatProvider(oauthplatform.WeChatConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			})
		}
	}
	oauthService := oauthapp.NewOAuthService(userRepo, oauthinfra.NewMySQLBindingRepository(dbConn), jwtUtil, sessionStore, providers, cfg.Auth.AccessExpireMin)
	oauthModule := oauth.New(oauthService)
```

- [ ] **Step 4: 修改 bootstrap.go / main.go**

在 `Bootstrap` 中把 `oauthModule` 加入 modules 列表（跟随现有模块注册模式）。

- [ ] **Step 5: 更新配置文件**

`configs/app.yaml` 加：

```yaml
oauth:
  providers:
    google:
      client_id: ""
      client_secret: ""
      redirect_url: "http://localhost:8080/api/v1/oauth/google/callback"
      enabled: false
    github:
      client_id: ""
      client_secret: ""
      redirect_url: "http://localhost:8080/api/v1/oauth/github/callback"
      enabled: false
```

- [ ] **Step 6: 验证编译**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 7: 更新 README.md**

特性列表、配置表、API 示例加 OAuth。

- [ ] **Step 8: 提交**

```bash
git add internal/config/config.go internal/config/validate.go internal/app/container.go internal/app/bootstrap.go cmd/server/main.go configs/app.yaml configs/app.prod.yaml internal/shared/errors/errors.go README.md
git commit -m "feat(oauth): wire module into app"
```

---

### Self-Review 记录

**1. Spec 覆盖：**
- Provider 抽象 + Google/GitHub/WeChat → Task 1
- 绑定表（方案 B）→ Task 2
- 登录流程（AuthURL → callback → Exchange → 匹配/创建用户 → 签发 JWT）→ Task 2 service
- 配置 `oauth.providers` → Task 3
- 错误码 `3xxx` → Task 3

**2. Placeholder 扫描：** 无 TBD/TODO。微信实现简化（`token.Extra("openid")`），标记为已知简化。

**3. Type 一致性：**
- `Provider`/`UserInfo` 接口 Task 1 定义，Task 2 service 引用一致。
- `OAuthBinding`/`BindingRepository` Task 2 定义，service/infra 引用一致。
- `NewOAuthService` 签名 Task 2 定义，Task 3 装配引用一致。
- `domain.TokenPair` 引用现有 auth domain。
