package oauth

import (
	"strings"
	"testing"

	"jimu/internal/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type viperSection struct{ v *viper.Viper }

func (s viperSection) UnmarshalKey(key string, rawVal any) error {
	return s.v.UnmarshalKey(key, rawVal)
}

func yamlSection(t *testing.T, yaml string) viperSection {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(yaml)))
	return viperSection{v: v}
}

func TestConfigKeyIsStable(t *testing.T) {
	assert.Equal(t, "oauth", ConfigKey, "对外配置键不得变化")
}

// TestDescriptorDeclaresConfigSection 描述符必须声明本段，否则框架不会加载/校验它。
func TestDescriptorDeclaresConfigSection(t *testing.T) {
	for _, spec := range Descriptor.Configs {
		if spec.Section == ConfigKey {
			require.NotNil(t, spec.New)
			_, ok := spec.New().(config.SectionConfig)
			require.True(t, ok, "段实例必须实现 config.SectionConfig")
			return
		}
	}
	t.Fatalf("Descriptor 必须声明配置段 %q", ConfigKey)
}

// loadConfig 走框架同款机制（config.LoadSection：解码 → 默认值 → 校验）。
func loadConfig(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// TestValidateProviders 迁移自 internal/config 的 TestValidateOAuthProviders，语义逐条不变。
func TestValidateProviders(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantErr bool
	}{
		{"未启用时忽略空配置", ProviderConfig{Enabled: false}, false},
		{"启用但缺 client_id", ProviderConfig{Enabled: true, RedirectURL: "https://x/cb"}, true},
		{"启用但缺 redirect_url", ProviderConfig{Enabled: true, ClientID: "id"}, true},
		{"内置提供商合法", ProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb"}, false},
		{"OIDC issuer 合法", ProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb", IssuerURL: "https://idp.example.com/realms/acme"}, false},
		{"OIDC issuer 非绝对地址", ProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb", IssuerURL: "idp.example.com"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Config{Providers: map[string]ProviderConfig{"p": tt.cfg}}.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestLoadMapsProviderKeys YAML 键名与下沉前一致。
func TestLoadMapsProviderKeys(t *testing.T) {
	cfg, err := loadConfig(yamlSection(t, `
oauth:
  providers:
    google:
      client_id: "gid"
      client_secret: "gsec"
      redirect_url: "https://x/cb"
      issuer_url: "https://accounts.google.com"
      scopes: ["openid", "email"]
      enabled: true
`))
	require.NoError(t, err)
	require.Contains(t, cfg.Providers, "google")
	p := cfg.Providers["google"]
	assert.Equal(t, "gid", p.ClientID)
	assert.Equal(t, "gsec", p.ClientSecret)
	assert.Equal(t, "https://x/cb", p.RedirectURL)
	assert.Equal(t, "https://accounts.google.com", p.IssuerURL)
	assert.Equal(t, []string{"openid", "email"}, p.Scopes)
	assert.True(t, p.Enabled)
}

// TestLoadRejectsInvalidProvider Load 内含校验（启用但缺必填项）。
func TestLoadRejectsInvalidProvider(t *testing.T) {
	_, err := loadConfig(yamlSection(t, "oauth:\n  providers:\n    p:\n      enabled: true\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oauth.providers.p")
}
