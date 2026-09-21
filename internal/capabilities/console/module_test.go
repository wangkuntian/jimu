package console

import (
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModuleNameAndContract 能力名与契约实现。
func TestModuleNameAndContract(t *testing.T) {
	m := New("test", "dev", nil, nil, nil, nil)
	assert.Equal(t, "console", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
}

// TestModuleDescriptor 描述符：依赖 auth/access，自管挂载，声明管理端通配权限点。
func TestModuleDescriptor(t *testing.T) {
	m := New("test", "dev", nil, nil, nil, nil)
	d := m.Descriptor()
	assert.Equal(t, "console", d.Name)
	assert.Equal(t, contract.MountSelfManaged, d.Normalized())
	assert.ElementsMatch(t, []string{"auth", "access"}, d.Requires)
	require.Len(t, d.Permissions, 4, "管理端通配权限点 GET/POST/PUT/DELETE")
	for _, p := range d.Permissions {
		assert.Equal(t, "/api/v1/admin/*", p.Resource)
	}
}

// TestModuleRegisterHTTP 平台级视图路由全部注册在 /api/v1/admin 下且恰好一次。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New("test", "dev", nil, nil, nil, nil)
	r := gin.New()
	m.RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"GET /api/v1/admin/error-codes",
		"GET /api/v1/admin/monitoring/status",
		"GET /api/v1/admin/monitoring/health",
		"GET /api/v1/admin/monitoring/metrics",
		"GET /api/v1/admin/ratelimit/auth",
		"GET /api/v1/admin/config",
		"PUT /api/v1/admin/config/:key",
		"POST /api/v1/admin/config/reload",
		"GET /api/v1/admin/ws",
		"POST /api/v1/admin/ws/push",
		"GET /api/v1/admin/ws/presence/:userId",
		"GET /api/v1/admin/ws/online",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}

	// 未配置 JWT 时 ws handler 返回 500 而非 panic
	m.initWS()
	assert.NotNil(t, m.wsHandler())
}

// TestModuleRegisterHTTPWithIPAllowlist IP 白名单注入后仍可注册（不 panic）。
func TestModuleRegisterHTTPWithIPAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	allow := func(c *gin.Context) { c.Next() }
	m := New("test", "dev", nil, nil, nil, nil, gin.HandlerFunc(allow))
	r := gin.New()
	m.RegisterHTTP(r)
	assert.NotEmpty(t, r.Routes())
}
