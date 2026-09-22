// internal/capabilities/oauth/module.go
package oauth

import (
	"embed"
	authmodule "jimu/internal/capabilities/auth"
	oauthapp "jimu/internal/capabilities/oauth/application"
	oauthinfra "jimu/internal/capabilities/oauth/infrastructure"
	"jimu/internal/capabilities/oauth/interfaces"
	oauthplatform "jimu/internal/capabilities/oauth/provider"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/httpclient"

	redistore "jimu/internal/kernel/redis"

	"gorm.io/gorm"
)

// Module OAuth 模块
type Module struct {
	service *oauthapp.OAuthService
}

// New 创建 OAuth 模块（自包含装配依赖）。authCfg 为 auth 能力配置（oauth.Requires 含 auth，方向合法）。
func New(db *gorm.DB, rdb redistore.Client, oauthCfg Config, authCfg authmodule.Config, httpClient *httpclient.Client) *Module {
	bindingRepo := oauthinfra.NewMySQLBindingRepository(db)
	jwtUtil := auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay)
	sessionStore := auth.NewRedisSessionStore(rdb)
	service := oauthapp.NewOAuthService(bindingRepo, jwtUtil, sessionStore, buildProviders(oauthCfg, httpClient), rdb, db, authCfg.AccessExpireMin)
	return &Module{service: service}
}

// buildProviders 按配置构造启用的 OAuth 提供商
func buildProviders(cfg Config, client *httpclient.Client) map[string]oauthplatform.Provider {
	providers := make(map[string]oauthplatform.Provider)
	for name, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}
		// 配了 issuer_url 即通用 OIDC（Keycloak/Okta/Auth0/Azure AD 等），provider 名自定义
		if pc.IssuerURL != "" {
			providers[name] = oauthplatform.NewOIDCProvider(name, oauthplatform.OIDCConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
				IssuerURL: pc.IssuerURL, Scopes: pc.Scopes,
			}, client)
			continue
		}
		switch name {
		case "google":
			providers[name] = oauthplatform.NewGoogleProvider(oauthplatform.GoogleConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			}, client)
		case "github":
			providers[name] = oauthplatform.NewGitHubProvider(oauthplatform.GitHubConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			}, client)
		case "wechat":
			providers[name] = oauthplatform.NewWeChatProvider(oauthplatform.WeChatConfig{
				ClientID: pc.ClientID, ClientSecret: pc.ClientSecret, RedirectURL: pc.RedirectURL,
			}, client)
		}
	}
	return providers
}

// Name 模块名
func (m *Module) Name() string { return "oauth" }

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明 OAuth 能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "oauth",
	Migrations: migrationsFS,
	Owns:       []string{"user_oauth_bindings"},
	Requires:   []string{"auth", "user"},
	Mount:      contract.MountPublic,
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

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
