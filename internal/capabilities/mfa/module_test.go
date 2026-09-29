package mfa

import (
	"testing"

	mfaapp "jimu/internal/capabilities/mfa/application"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newMFAModule(t *testing.T) *Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return New(db, Config{
		JWTSecret:         "01234567890123456789012345678901",
		Issuer:            "jimu",
		AccessExpireMin:   30,
		RefreshExpireDay:  7,
		TrustedDeviceDays: 30,
	}, nil)
}

func TestModuleNameAndContract(t *testing.T) {
	m := newMFAModule(t)
	assert.Equal(t, "mfa", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.NotNil(t, m.Service())
	var _ contract.MFAVerifier = m.Service()
}

// TestModuleDescriptor 描述符：依赖 user，自管挂载，声明 MFA + 设备权限点。
func TestModuleDescriptor(t *testing.T) {
	d := newMFAModule(t).Descriptor()
	assert.Equal(t, "mfa", d.Name)
	assert.Equal(t, contract.MountSelfManaged, d.Normalized())
	assert.ElementsMatch(t, []string{"user"}, d.Requires)
	assert.Equal(t, []string{"auth"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"user_mfa", "trusted_devices"}, d.Owns)
	require.NotNil(t, d.Migrations)
	// 3 条 MFA + 3 条设备
	assert.Len(t, d.Permissions, 6)
}

// TestModuleRegisterHTTP MFA 与可信设备路由全部注册且恰好一次。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	newMFAModule(t).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /api/v1/auth/mfa/setup",
		"POST /api/v1/auth/mfa/enable",
		"POST /api/v1/auth/mfa/disable",
		"GET /api/v1/auth/devices",
		"DELETE /api/v1/auth/devices",
		"DELETE /api/v1/auth/devices/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}

var _ = mfaapp.MFAService{}
