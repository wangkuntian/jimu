package passkey

import (
	"testing"

	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestModuleNameAndDescriptor 能力名与描述符（WebAuthn 关闭时仍注册凭证路由）。
func TestModuleNameAndDescriptor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	m := New(Deps{DB: db, AuthCfg: authmodule.Config{Issuer: "jimu"}})

	assert.Equal(t, "passkey", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)

	d := m.Descriptor()
	assert.Equal(t, "passkey", d.Name)
	assert.Equal(t, contract.MountSelfManaged, d.Normalized())
	assert.NotNil(t, d.Migrations)
}

// TestModuleRegisterHTTP WebAuthn 路由全部注册且恰好一次。
func TestModuleRegisterHTTP(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(Deps{DB: db, AuthCfg: authmodule.Config{Issuer: "jimu"}}).RegisterHTTP(r)

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
