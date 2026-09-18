package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminapi "jimu/internal/capabilities/admin/domain"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTestDB(t)
	fullKey := "jimu_" + "abcdef0123456789abcdef0123456789"
	createKey(t, db, &adminapi.APIKey{
		TenantID: 7,
		Name:     "service-a",
		KeyHash:  HashKey(fullKey),
		Enabled:  true,
	})
	verifier := NewAPIKeyVerifier(NewDBAPIKeyStore(db))

	r := gin.New()
	r.Use(APIKeyAuthMiddleware(verifier))
	r.GET("/internal", func(c *gin.Context) {
		key, ok := c.Get("api_key")
		require.True(t, ok)
		ak, ok := key.(*APIKey)
		require.True(t, ok)
		assert.Equal(t, "service-a", ak.Name)
		// 租户来自 Key 自身归属，同时注入 gin context 与 request context
		assert.Equal(t, uint64(7), ak.TenantID)
		assert.Equal(t, uint64(7), tenant.FromContext(c.Request.Context()))
		v, exists := c.Get("tenant_id")
		require.True(t, exists)
		assert.Equal(t, uint64(7), v)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("missing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal", nil)
		req.Header.Set(APIKeyHeader, "jimu_wrong")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal", nil)
		req.Header.Set(APIKeyHeader, fullKey)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyAuthMiddlewareWithoutTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTestDB(t)
	fullKey := "jimu_" + "ffffffffffffffffffffffffffffffff"
	createKey(t, db, &adminapi.APIKey{
		Name:    "legacy",
		KeyHash: HashKey(fullKey),
		Enabled: true,
	})
	verifier := NewAPIKeyVerifier(NewDBAPIKeyStore(db))

	r := gin.New()
	r.Use(APIKeyAuthMiddleware(verifier))
	r.GET("/internal", func(c *gin.Context) {
		// 未归属租户（tenant_id=0）时不注入，保持平台级视角
		assert.Equal(t, uint64(0), tenant.FromContext(c.Request.Context()))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal", nil)
	req.Header.Set(APIKeyHeader, fullKey)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTestDB(t)
	scopedKey := "jimu_" + "11111111111111111111111111111111"
	wildcardKey := "jimu_" + "22222222222222222222222222222222"
	emptyKey := "jimu_" + "33333333333333333333333333333333"
	createKey(t, db, &adminapi.APIKey{TenantID: 1, Name: "scoped", KeyHash: HashKey(scopedKey), Scopes: `["user:read"]`, Enabled: true})
	createKey(t, db, &adminapi.APIKey{TenantID: 1, Name: "wildcard", KeyHash: HashKey(wildcardKey), Scopes: `["*"]`, Enabled: true})
	createKey(t, db, &adminapi.APIKey{TenantID: 1, Name: "empty", KeyHash: HashKey(emptyKey), Enabled: true})
	verifier := NewAPIKeyVerifier(NewDBAPIKeyStore(db))

	r := gin.New()
	r.GET("/scoped", APIKeyAuthMiddleware(verifier), RequireScope("user:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/other", APIKeyAuthMiddleware(verifier), RequireScope("user:write"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	// 未挂载认证中间件：RequireScope 应拒绝
	r.GET("/bare", RequireScope("user:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do := func(path, key string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if key != "" {
			req.Header.Set(APIKeyHeader, key)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, do("/scoped", scopedKey), "具备 scope 应放行")
	assert.Equal(t, http.StatusOK, do("/scoped", wildcardKey), "通配 * 应放行")
	assert.Equal(t, http.StatusForbidden, do("/scoped", emptyKey), "空 scopes 应拒绝一切")
	assert.Equal(t, http.StatusForbidden, do("/other", scopedKey), "缺少 scope 应拒绝")
	assert.Equal(t, http.StatusUnauthorized, do("/bare", scopedKey), "未认证时 RequireScope 应拒绝")
	assert.Equal(t, http.StatusUnauthorized, do("/scoped", ""), "缺少 API Key 应拒绝")
}
