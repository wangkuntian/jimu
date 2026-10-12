package apikey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "jimu/internal/capabilities/apikey/domain"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A2.1 已过期的 Key 必须被拒绝（有效期内才产生归属）。
func TestA2_1_ExpiredAndDisabledKeysAreRejected(t *testing.T) {
	db := newTestDB(t)
	v := NewAPIKeyVerifier(NewDBAPIKeyStore(db))
	ctx := context.Background()

	expired := "jimu_" + "aaaa1111aaaa1111aaaa1111aaaa1111"
	createKey(t, db, &domain.APIKey{Name: "expired", KeyHash: HashKey(expired), Enabled: true, ExpiresAt: time.Now().Add(-time.Hour)})
	got, err := v.Verify(ctx, expired)
	require.Error(t, err, "已过期 Key 必须拒绝")
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "expired")

	disabled := "jimu_" + "bbbb2222bbbb2222bbbb2222bbbb2222"
	createKey(t, db, &domain.APIKey{Name: "disabled", KeyHash: HashKey(disabled), Enabled: false})
	got, err = v.Verify(ctx, disabled)
	require.Error(t, err, "已禁用 Key 必须拒绝")
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "disabled")
}

// A2.3 伪造前缀/过短格式必须被拒绝。
// 故意让库里存在一条 **hash 命中** 的记录：只有前缀/长度校验才能拦住它，
// 这样这条断言就不是空转（Step 5 用放宽实现演示它确实变红）。
func TestA2_3_ForgedKeyFormatIsRejectedBeforeLookup(t *testing.T) {
	db := newTestDB(t)
	v := NewAPIKeyVerifier(NewDBAPIKeyStore(db))
	ctx := context.Background()

	forged := "forged-abcdef0123456789"
	createKey(t, db, &domain.APIKey{Name: "forged", TenantID: 5, KeyHash: HashKey(forged), Enabled: true})
	got, err := v.Verify(ctx, forged)
	require.Error(t, err, "缺少 jimu_ 前缀的 Key 必须拒绝")
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "invalid key format")

	short := "jimu_short"
	createKey(t, db, &domain.APIKey{Name: "short", TenantID: 5, KeyHash: HashKey(short), Enabled: true})
	got, err = v.Verify(ctx, short)
	require.Error(t, err, "前缀合法但过短的 Key 必须拒绝")
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "too short")
}

// T2.4 API Key 路径：FromContext == 该 Key 的归属租户；伪造 header/query 不生效。
// 过期 Key 走「整请求被拒」，业务 handler 不可达 ⇒ 不产生任何租户归属。
func TestT2_4_KeyTenantWinsOverForgedInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	v := NewAPIKeyVerifier(NewDBAPIKeyStore(db))

	fullKey := "jimu_" + "beefbeefbeefbeefbeefbeefbeefbeef"
	createKey(t, db, &domain.APIKey{TenantID: 7, Name: "svc", KeyHash: HashKey(fullKey), Enabled: true})

	var gotTenant uint64
	var reached bool
	r := gin.New()
	r.Use(APIKeyAuthMiddleware(v))
	r.GET("/internal", func(c *gin.Context) {
		reached = true
		gotTenant = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// ① 有效 Key + 伪造 header/query → 归属仍取自 Key 自身
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal?tenant_id=9&tid=9", nil)
	req.Header.Set(APIKeyHeader, fullKey)
	req.Header.Set("X-Tenant-ID", "9")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, reached)
	assert.Equal(t, uint64(7), gotTenant, "T2.4 租户必须来自已校验 Key 的归属")

	// ② 过期 Key + 伪造 header → 整请求被拒，handler 不可达
	expiredKey := "jimu_" + "cafecafecafecafecafecafecafecafe"
	createKey(t, db, &domain.APIKey{TenantID: 7, Name: "expired", KeyHash: HashKey(expiredKey), Enabled: true, ExpiresAt: time.Now().Add(-time.Minute)})
	reached, gotTenant = false, 0
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/internal?tenant_id=9", nil)
	req.Header.Set(APIKeyHeader, expiredKey)
	req.Header.Set("X-Tenant-ID", "9")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, reached, "过期 Key 必须整请求被拒绝")
	assert.Equal(t, uint64(0), gotTenant)
}
