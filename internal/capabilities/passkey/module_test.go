package passkey

import (
	"strings"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestModuleNameAndDescriptor 能力名与描述符（WebAuthn 关闭时仍注册凭证路由）。
func TestModuleNameAndDescriptor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	m := New(Deps{DB: db, AuthCfg: contract.AuthConfig{Issuer: "jimu"}, Config: Config{}})

	assert.Equal(t, "passkey", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)

	d := m.Descriptor()
	assert.Equal(t, "passkey", d.Name)
	assert.Equal(t, contract.MountSelfManaged, d.Normalized())
	assert.NotNil(t, d.Migrations)
}

func TestDescriptorDeclaresOwnConfiguration(t *testing.T) {
	require.Len(t, Descriptor.Configs, 1)
	assert.Equal(t, "passkey", Descriptor.Configs[0].Section)
	_, ok := Descriptor.Configs[0].New().(config.SectionConfig)
	assert.True(t, ok)
}

func TestConfigValidatesWebAuthnSettings(t *testing.T) {
	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{SessionTTLMin: -1}).Validate(), "disabled passkey config skips its validation")

	valid := &Config{
		Enabled:       true,
		RPDisplayName: "Jimu",
		RPID:          "example.com",
		RPOrigins:     []string{"https://example.com"},
	}
	assert.NoError(t, valid.Validate())

	for _, tc := range []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "missing RP ID", cfg: Config{Enabled: true}, want: "rp_id"},
		{name: "missing origins", cfg: Config{Enabled: true, RPID: "example.com"}, want: "rp_origins"},
		{name: "relative origin", cfg: Config{Enabled: true, RPID: "example.com", RPOrigins: []string{"/relative"}}, want: "absolute http(s) origin"},
		{name: "negative session TTL", cfg: Config{Enabled: true, RPID: "example.com", RPOrigins: []string{"https://example.com"}, SessionTTLMin: -1}, want: "session_ttl_min"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestConfigDecodesPasskeySection(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader("passkey:\n  enabled: true\n  rp_id: example.com\n  rp_origins: [https://example.com]\n")))
	var cfg Config
	require.NoError(t, v.UnmarshalKey(ConfigKey, &cfg))
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "example.com", cfg.RPID)
	assert.Equal(t, []string{"https://example.com"}, cfg.RPOrigins)
}

// TestModuleRegisterHTTP WebAuthn 路由全部注册且恰好一次。
func TestModuleRegisterHTTP(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(Deps{DB: db, AuthCfg: contract.AuthConfig{Issuer: "jimu"}, Config: Config{}}).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /api/v1/auth/webauthn/login/begin",
		"POST /api/v1/auth/webauthn/login/finish",
		"POST /api/v1/auth/webauthn/register/begin",
		"POST /api/v1/auth/webauthn/register/finish",
		"GET /api/v1/auth/webauthn/credentials",
		"PUT /api/v1/auth/webauthn/credentials/:id",
		"DELETE /api/v1/auth/webauthn/credentials/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}
