// internal/capabilities/oauth/module_test.go
package oauth

import (
	"testing"

	oauthplatform "jimu/internal/capabilities/oauth/provider"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/httpclient"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestModule() *Module {
	httpClient := httpclient.New(httpclient.Config{})
	return New(
		nil, // db：仅装配，模块方法未使用
		&redis.Client{},
		Config{
			Providers: map[string]ProviderConfig{
				"google": {ClientID: "g-id", ClientSecret: "g-secret", RedirectURL: "https://x/g", Enabled: true},
				"github": {ClientID: "h-id", Enabled: true},
			},
		},
		config.AuthConfig{JWTSecret: "01234567890123456789012345678901", Issuer: "jimu", AccessExpireMin: 30, RefreshExpireDay: 7},
		httpClient,
	)
}

func TestModuleNameAndContract(t *testing.T) {
	m := newTestModule()
	assert.Equal(t, "oauth", m.Name())

	var _ contract.Module = m

	// 无定时任务/事件注册，不得 panic
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
}

func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := newTestModule()
	r := gin.New()

	// *gin.RouterGroup 满足 contract.Router
	var router contract.Router = r.Group("/")
	m.RegisterHTTP(router)

	routes := make(map[string]bool)
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes["GET /api/v1/oauth/:provider/login"], "缺少 login 路由")
	assert.True(t, routes["GET /api/v1/oauth/:provider/callback"], "缺少 callback 路由")
}

func TestBuildProvidersFiltersEnabled(t *testing.T) {
	httpClient := httpclient.New(httpclient.Config{})
	providers := buildProviders(Config{
		Providers: map[string]ProviderConfig{
			"google":  {ClientID: "g-id", Enabled: true},
			"github":  {ClientID: "h-id", Enabled: false}, // 禁用应被过滤
			"wechat":  {ClientID: "w-id", Enabled: true},
			"unknown": {ClientID: "u-id", Enabled: true}, // 未支持名称应被忽略
		},
	}, httpClient)

	require.Len(t, providers, 2)
	assert.Contains(t, providers, "google")
	assert.Contains(t, providers, "wechat")
	_, hasGithub := providers["github"]
	assert.False(t, hasGithub)

	assert.Equal(t, "google", providers["google"].Name())
	assert.Equal(t, "wechat", providers["wechat"].Name())
}

func TestBuildProvidersUsesOIDCWhenIssuerConfigured(t *testing.T) {
	httpClient := httpclient.New(httpclient.Config{})
	providers := buildProviders(Config{
		Providers: map[string]ProviderConfig{
			// issuer_url 非空即按通用 OIDC 处理，provider 名自定义
			"keycloak": {ClientID: "k-id", IssuerURL: "https://idp.example.com/realms/acme", RedirectURL: "https://x/k", Enabled: true},
			// 未启用不构造
			"okta": {ClientID: "o-id", IssuerURL: "https://okta.example.com", Enabled: false},
		},
	}, httpClient)

	require.Len(t, providers, 1)
	require.Contains(t, providers, "keycloak")
	assert.Equal(t, "keycloak", providers["keycloak"].Name())
	assert.IsType(t, &oauthplatform.OIDCProvider{}, providers["keycloak"])
}
