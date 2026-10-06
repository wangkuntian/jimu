package authmodule

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() Config {
	return Config{
		JWTSecret:             strings.Repeat("x", 32),
		Issuer:                "jimu",
		AccessExpireMin:       30,
		RefreshExpireDay:      7,
		LoginRateLimit:        10,
		LoginRateWindowSec:    60,
		RegisterRateLimit:     5,
		RegisterRateWindowSec: 300,
		ResetCodeTTLMin:       15,
	}
}

func TestPortViewContainsOnlyAuthConfiguration(t *testing.T) {
	cfg := validConfig()
	view := cfg.PortView()
	assert.Equal(t, cfg.JWTSecret, view.JWTSecret)
	assert.Equal(t, cfg.PublicRegistration, view.PublicRegistration)
	fields := make([]string, reflect.TypeOf(view).NumField())
	for i := range fields {
		fields[i] = reflect.TypeOf(view).Field(i).Name
	}
	assert.NotContains(t, fields, "WebAuthn")
	assert.NotContains(t, fields, "Provisioning")
}

func TestValidateCommonAuthChecks(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		key    string
	}{
		{"empty issuer", func(c *Config) { c.Issuer = "" }, "invalid auth configuration"},
		{"zero access expiry", func(c *Config) { c.AccessExpireMin = 0 }, "invalid auth configuration"},
		{"zero refresh expiry", func(c *Config) { c.RefreshExpireDay = 0 }, "invalid auth configuration"},
		{"zero reset ttl", func(c *Config) { c.ResetCodeTTLMin = 0 }, "reset_code_ttl_min"},
		{"zero login rate limit", func(c *Config) { c.LoginRateLimit = 0 }, "invalid auth rate limit"},
		{"zero register window", func(c *Config) { c.RegisterRateWindowSec = 0 }, "invalid auth rate limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.key)
		})
	}

	base := validConfig()
	assert.NoError(t, base.Validate())
}

// TestValidateProdRejectsWeakJWTSecret 生产加严：弱/占位/未展开的 jwt_secret 一律拒绝。
func TestValidateProdRejectsWeakJWTSecret(t *testing.T) {
	for _, secret := range []string{
		"",
		"short",
		"change-me-in-production",
		"${JWT_SECRET}",
		strings.Repeat("a", 31),
	} {
		cfg := validConfig()
		cfg.JWTSecret = secret
		assert.Error(t, cfg.ValidateProd(), "secret %q 在生产环境必须被拒绝", secret)
	}
}

func TestAuthConfigPortViewDoesNotExposeOtherCapabilitySettings(t *testing.T) {
	typ := reflect.TypeOf(contract.AuthConfig{})
	for _, name := range []string{"WebAuthn", "Provisioning"} {
		_, ok := typ.FieldByName(name)
		assert.False(t, ok, "contract.AuthConfig must not expose %s", name)
	}
}

func TestValidateProdAcceptsStrongJWTSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWTSecret = strings.Repeat("a", 32)
	assert.NoError(t, cfg.ValidateProd())
}

// TestValidateProdDoesNotLeakSecret 校验错误不得回显密钥本身。
func TestValidateProdDoesNotLeakSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWTSecret = "super-secret-but-short"
	err := cfg.ValidateProd()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), cfg.JWTSecret)
}

// TestApplyDefaultsJWTSecretEnvOverride JWT_SECRET 覆盖 YAML 值（原 config.applyEnvOverrides 行为）。
func TestApplyDefaultsJWTSecretEnvOverride(t *testing.T) {
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("JWT_PREVIOUS_SECRET_FILE", "")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))

	cfg := Config{JWTSecret: "from-yaml"}
	cfg.ApplyDefaults()
	assert.Equal(t, strings.Repeat("a", 32), cfg.JWTSecret)
}

// TestApplyDefaultsJWTPreviousSecretFromFile 支持 _FILE 后缀（Docker Secrets 兼容）。
func TestApplyDefaultsJWTPreviousSecretFromFile(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("JWT_PREVIOUS_SECRET", "")

	path := filepath.Join(t.TempDir(), "previous-secret")
	require.NoError(t, os.WriteFile(path, []byte("rotated-secret\n"), 0o600))
	t.Setenv("JWT_PREVIOUS_SECRET_FILE", path)

	cfg := Config{}
	cfg.ApplyDefaults()
	assert.Equal(t, "rotated-secret", cfg.JWTPreviousSecret, "应去除 secret 文件末尾换行")
}

// TestApplyDefaultsKeepsYAMLWhenEnvUnset 环境变量未提供时保留 YAML 值。
func TestApplyDefaultsKeepsYAMLWhenEnvUnset(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("JWT_PREVIOUS_SECRET", "")
	t.Setenv("JWT_PREVIOUS_SECRET_FILE", "")

	cfg := Config{JWTSecret: "from-yaml", JWTPreviousSecret: "old-from-yaml"}
	cfg.ApplyDefaults()
	assert.Equal(t, "from-yaml", cfg.JWTSecret)
	assert.Equal(t, "old-from-yaml", cfg.JWTPreviousSecret)
}

// TestDescriptorDeclaresOwnConfigSection 描述符声明 auth 段，且段实例实现机制要求的钩子
// （ValidateProd 是 ProdConfigValidator，框架在 APP_ENV=prod 时按类型断言调用）。
func TestDescriptorDeclaresOwnConfigSection(t *testing.T) {
	require.Len(t, Descriptor.Configs, 1)
	assert.Equal(t, ConfigKey, Descriptor.Configs[0].Section)

	v := Descriptor.Configs[0].New()
	_, ok := v.(config.SectionConfig)
	assert.True(t, ok, "auth 段必须实现 config.SectionConfig")
	_, prodOK := v.(config.ProdConfigValidator)
	assert.True(t, prodOK, "auth 段必须实现 config.ProdConfigValidator（prod jwt_secret 加严）")
}
