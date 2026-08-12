package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminapi "jimu/internal/modules/admin/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTestDB(t)
	fullKey := "jimu_" + "abcdef0123456789abcdef0123456789"
	createKey(t, db, &adminapi.APIKey{
		Name:    "service-a",
		KeyHash: HashKey(fullKey),
		Enabled: true,
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
