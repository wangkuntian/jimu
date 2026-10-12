package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tenantProbeState 记录探测 handler 是否执行、以及它从 request context 读到的租户。
type tenantProbeState struct {
	tenant  uint64
	reached bool
}

// newJWTChainRouter 复刻 internal/capabilities/auth/interfaces/protected.go 的租户相关链路：
// auth.AuthMiddleware（验签并把 claim 注入 gin context）→ tenant.Middleware（写入 request context）。
func newJWTChainRouter(t *testing.T) (*gin.Engine, *auth.JWT, *tenantProbeState) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	j := auth.New(strings.Repeat("k", 32), "jimu", 30, 7)
	state := &tenantProbeState{}
	r := gin.New()
	r.Use(auth.AuthMiddleware(j))
	r.Use(tenant.Middleware())
	r.GET("/probe", func(c *gin.Context) {
		state.reached = true
		state.tenant = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	return r, j, state
}

// T2.3 JWT 路径：claim tid 存在时 FromContext == tid。
func TestT2_3_JWTClaimTenantReachesRequestContext(t *testing.T) {
	r, j, state := newJWTChainRouter(t)
	token, err := j.GenerateAccess(42, 5, "sid-1")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, state.reached)
	assert.Equal(t, uint64(5), state.tenant, "T2.3 FromContext 必须等于已验签 claim 的 tid")
}

// T2.1/T2.2 伪造租户 header 与 query 参数不得覆盖已验签 claim 的 tid。
func TestT2_1_T2_2_ForgedHeaderAndQueryDoNotOverrideClaim(t *testing.T) {
	r, j, state := newJWTChainRouter(t)
	token, err := j.GenerateAccess(42, 5, "sid-1")
	require.NoError(t, err)

	cases := []struct {
		name        string
		target      string
		forgedHeads map[string]string
	}{
		{"伪造 header", "/probe", map[string]string{"X-Tenant-ID": "9", "tenant_id": "9", "tid": "9"}},
		{"伪造 query", "/probe?tenant_id=9&tid=9&tenant=9", nil},
		{"同时伪造 header 与 query", "/probe?tenant_id=9", map[string]string{"X-Tenant-ID": "9"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state.reached = false
			state.tenant = 0
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			for k, v := range tc.forgedHeads {
				req.Header.Set(k, v)
			}
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.True(t, state.reached)
			assert.Equal(t, uint64(5), state.tenant, "T2.1/T2.2 伪造的 header/query 不得生效")
		})
	}
}

// T2.1/T2.2（无凭据形态）只带伪造 header/query 的请求必须被拒绝，探测 handler 不可达。
func TestT2_1_T2_2_ForgedHeaderAndQueryWithoutCredentials(t *testing.T) {
	r, _, state := newJWTChainRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe?tenant_id=9", nil)
	req.Header.Set("X-Tenant-ID", "9")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, state.reached, "未验签的请求不得进入业务 handler")
	assert.Equal(t, uint64(0), state.tenant)
}

// T2.5 无任何凭据：验签中间件整请求拒绝，不会产生任何租户归属。
func TestT2_5_NoCredentialsRequestIsRejected(t *testing.T) {
	r, _, state := newJWTChainRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, state.reached, "无凭据请求必须被拒绝")
	assert.Equal(t, uint64(0), state.tenant, "T2.5 不产生租户归属（平台级视角在纯逻辑侧由 kernel/tenant 断言）")
}

// newTenantLinkRouter 只挂载真实的 tenant.Middleware —— 即「gin context → request context」
// 这一步跨中间件链路本身。上面的无凭据用例在验签层就被拒绝，handler 不可达，
// 因此无法观测链路是否把客户端标识当成了租户来源；这里让请求真正到达 handler 并读取 FromContext。
func newTenantLinkRouter(t *testing.T) (*gin.Engine, *tenantProbeState) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	state := &tenantProbeState{}
	r := gin.New()
	r.Use(tenant.Middleware())
	r.GET("/probe", func(c *gin.Context) {
		state.reached = true
		state.tenant = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	return r, state
}

// T2.1/T2.2（无凭据 + 仅伪造标识，本任务承接的 T2.6 覆盖）：客户端 header/query
// 不得成为租户来源，即使请求在真实 tenant.Middleware 链路上到达业务 handler，
// FromContext 也必须为 0。伪造 header => 0 这一格在此被真实断言。
func TestT2_1_T2_2_ForgedHeaderAndQueryAloneYieldNoTenant(t *testing.T) {
	r, state := newTenantLinkRouter(t)

	cases := []struct {
		name        string
		target      string
		forgedHeads map[string]string
	}{
		{"仅伪造 header", "/probe", map[string]string{"X-Tenant-ID": "9", "tenant_id": "9", "tid": "9"}},
		{"仅伪造 query", "/probe?tenant_id=9&tid=9&tenant=9", nil},
		{"同时伪造 header 与 query", "/probe?tenant=9", map[string]string{"X-Tenant-ID": "9"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state.reached = false
			state.tenant = 0
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			for k, v := range tc.forgedHeads {
				req.Header.Set(k, v)
			}
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.True(t, state.reached, "探测 handler 必须可达，否则断言无检测力")
			assert.Equal(t, uint64(0), state.tenant,
				"T2.1/T2.2 无凭据 + 仅伪造 header/query：FromContext 必须为 0，不得产生租户归属")
		})
	}
}
