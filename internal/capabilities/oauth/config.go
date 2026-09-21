package oauth

import (
	"fmt"
	"net/url"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "oauth"

// ProviderConfig 单个 OAuth 提供商配置（原 config.OAuthProviderConfig，P2.1 下沉）。
// 填了 issuer_url 的提供商按通用 OIDC 处理（provider 名可自定义，如 keycloak/okta/azuread）。
type ProviderConfig struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	IssuerURL    string   `mapstructure:"issuer_url"`
	Scopes       []string `mapstructure:"scopes"`
	Enabled      bool     `mapstructure:"enabled"`
}

// Config OAuth 登录配置段（原 config.OAuthConfig，P2.1 下沉）。
type Config struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// ApplyDefaults 本能力无配置层默认值（scopes 缺省由提供商实现处理）。
func (c *Config) ApplyDefaults() {}

// Validate 校验本能力配置段。仅在本能力启用时由组合根调用（设计 §8）。
// 迁移自 internal/config 的 validateOAuthProviders，语义不变。
func (c Config) Validate() error {
	for name, p := range c.Providers {
		if !p.Enabled {
			continue
		}
		if p.ClientID == "" || p.RedirectURL == "" {
			return fmt.Errorf("oauth.providers.%s requires client_id and redirect_url when enabled", name)
		}
		if p.IssuerURL == "" {
			continue
		}
		u, err := url.Parse(p.IssuerURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("oauth.providers.%s.issuer_url must be an absolute http(s) URL", name)
		}
	}
	return nil
}

// Load 解码并校验本能力配置段。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
