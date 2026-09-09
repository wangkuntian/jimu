package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminapi "jimu/internal/modules/admin/domain"
	"jimu/internal/platform/tenant"

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
