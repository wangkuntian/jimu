// internal/modules/oauth/module.go
package oauth

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	oauthapp "jimu/internal/modules/oauth/application"
	oauthinfra "jimu/internal/modules/oauth/infrastructure"
	"jimu/internal/modules/oauth/interfaces"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module OAuth 模块
type Module struct {
	service *oauthapp.OAuthService
}

// New 创建 OAuth 模块（自包含装配依赖）
func New(db *gorm.DB, rdb *redis.Client, oauthCfg config.OAuthConfig, authCfg config.AuthConfig) *Module {
	bindingRepo := oauthinfra.NewMySQLBindingRepository(db)
	jwtUtil := auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay)
	sessionStore := auth.NewRedisSessionStore(rdb)
	service := oauthapp.NewOAuthService(bindingRepo, jwtUtil, sessionStore, buildProviders(oauthCfg), rdb, db, authCfg.AccessExpireMin)
	return &Module{service: service}
}

// buildProviders 按配置构造启用的 OAuth 提供商
func buildProviders(cfg config.OAuthConfig) map[string]oauthplatform.Provider {
	providers := make(map[string]oauthplatform.Provider)
	for name, pc := range cfg.Providers {
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
	return providers
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
