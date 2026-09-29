package apikey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "jimu/internal/capabilities/apikey/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAPIKeyModuleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestModuleNameAndContract(t *testing.T) {
	m := New(newAPIKeyModuleDB(t))
	assert.Equal(t, "apikey", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.Equal(t, "apikey", m.Descriptor().Name)
}

// TestModuleDescriptor 描述符：软依赖 tenant（可选配额），自有 api_keys 表。
func TestModuleDescriptor(t *testing.T) {
	d := New(newAPIKeyModuleDB(t)).Descriptor()
	assert.Equal(t, []string{"tenant"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"api_keys"}, d.Owns)
}

// TestModuleRegisterHTTP API Key 管理端点注册且恰好一次（原 admin 归属）。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(newAPIKeyModuleDB(t)).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	assert.Equal(t, 1, got["GET /api/v1/admin/apikeys"])
	assert.Equal(t, 1, got["POST /api/v1/admin/apikeys"])
	assert.Equal(t, 1, got["GET /api/v1/admin/apikeys/:id"])
	assert.Equal(t, 1, got["DELETE /api/v1/admin/apikeys/:id"])
}

// TestModuleWithQuota 注入配额校验不 panic。
func TestModuleWithQuota(t *testing.T) {
	m := New(newAPIKeyModuleDB(t), fakeAPIKeyQuota{})
	assert.Equal(t, "apikey", m.Name())
}

type fakeAPIKeyQuota struct{}

func (fakeAPIKeyQuota) CheckAPIKeyQuota(context.Context, uint64) error { return nil }

// TestModuleProtectedHTTPMiddleware machine 形态：apikey 是唯一受保护中间件提供者，
// 返回「X-API-Key 认证（含租户注入）+ scope 校验」链。
func TestModuleProtectedHTTPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	scopedKey := "jimu_" + "aaaa1111bbbb2222cccc3333dddd4444"
	wildcardKey := "jimu_" + "bbbb1111cccc2222dddd3333eeee4444"
	unscopedKey := "jimu_" + "cccc1111dddd2222eeee3333ffff4444"
	createKey(t, db, &domain.APIKey{TenantID: 7, Name: "machine", KeyHash: HashKey(scopedKey), Scopes: `["` + ScopeProtected + `"]`, Enabled: true})
	createKey(t, db, &domain.APIKey{TenantID: 7, Name: "wildcard", KeyHash: HashKey(wildcardKey), Scopes: `["*"]`, Enabled: true})
	createKey(t, db, &domain.APIKey{TenantID: 7, Name: "unscoped", KeyHash: HashKey(unscopedKey), Enabled: true})

	chain, err := New(db, WithProtectedMiddleware(true)).ProtectedHTTPMiddleware()
	require.NoError(t, err)
	require.Len(t, chain, 2, "链上应为认证 + scope 校验")

	r := gin.New()
	r.Use(chain...)
	r.GET("/protected", func(c *gin.Context) {
		// 租户只来自 Key 归属，与 JWT 路径的 tenant_id 约定一致
		assert.Equal(t, uint64(7), tenant.FromContext(c.Request.Context()))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do := func(key string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		if key != "" {
			req.Header.Set(APIKeyHeader, key)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, do(scopedKey), "持有受保护 scope 的 Key 应放行")
	assert.Equal(t, http.StatusOK, do(wildcardKey), "通配 * 应放行")
	assert.Equal(t, http.StatusUnauthorized, do(""), "缺少 API Key 应拒绝")
	assert.Equal(t, http.StatusUnauthorized, do("jimu_invalid"), "无效 API Key 应拒绝")
	assert.Equal(t, http.StatusForbidden, do(unscopedKey), "缺少 scope 应拒绝")
}

// TestModuleProtectedHTTPMiddlewareYieldsToAuth full 形态：auth 已是受保护中间件提供者，
// apikey 返回空链让位，避免触发 bootstrap 的单提供者 fail-closed 规则。
func TestModuleProtectedHTTPMiddlewareYieldsToAuth(t *testing.T) {
	chain, err := New(newAPIKeyModuleDB(t)).ProtectedHTTPMiddleware()
	require.NoError(t, err)
	assert.Empty(t, chain)
}
