package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ipAllowlistRouter(cidrs []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPAllowlist(cidrs))
	r.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func doFromIP(t *testing.T, r *gin.Engine, remoteAddr string) int {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.RemoteAddr = remoteAddr
	r.ServeHTTP(w, req)
	return w.Code
}

func TestIPAllowlistEmptyListAllowsAll(t *testing.T) {
	r := ipAllowlistRouter(nil)
	assert.Equal(t, http.StatusOK, doFromIP(t, r, "203.0.113.9:1234"))
}

func TestIPAllowlistCIDR(t *testing.T) {
	r := ipAllowlistRouter([]string{"10.0.0.0/8", "192.168.1.0/24"})

	assert.Equal(t, http.StatusOK, doFromIP(t, r, "10.1.2.3:1000"))
	assert.Equal(t, http.StatusOK, doFromIP(t, r, "192.168.1.55:1000"))
	assert.Equal(t, http.StatusForbidden, doFromIP(t, r, "192.168.2.55:1000"))
	assert.Equal(t, http.StatusForbidden, doFromIP(t, r, "203.0.113.9:1000"))
}

func TestIPAllowlistSingleIPEntry(t *testing.T) {
	r := ipAllowlistRouter([]string{"127.0.0.1"})
	assert.Equal(t, http.StatusOK, doFromIP(t, r, "127.0.0.1:1000"))
	assert.Equal(t, http.StatusForbidden, doFromIP(t, r, "127.0.0.2:1000"))
}

func TestIPAllowlistHandlesIPv4MappedIPv6(t *testing.T) {
	r := ipAllowlistRouter([]string{"10.0.0.0/8"})
	// gin 对 IPv4 请求通常返回 ::ffff:v4，需要在白名单中正确命中
	assert.Equal(t, http.StatusOK, doFromIP(t, r, "[::ffff:10.1.2.3]:1000"))
}

func TestIPAllowlistInvalidEntriesAreIgnored(t *testing.T) {
	// 非法条目被忽略；配置阶段已由 config.Validate 拦截，这里保证不会 panic
	r := ipAllowlistRouter([]string{"not-a-cidr", "10.0.0.0/8"})
	assert.Equal(t, http.StatusOK, doFromIP(t, r, "10.1.2.3:1000"))
	assert.Equal(t, http.StatusForbidden, doFromIP(t, r, "203.0.113.9:1000"))
}

func TestIPAllowlistAllInvalidDeniesAll(t *testing.T) {
	// 列表非空但无有效条目 → 不降级为放行（fail-closed）
	r := ipAllowlistRouter([]string{"not-a-cidr"})
	assert.Equal(t, http.StatusForbidden, doFromIP(t, r, "10.1.2.3:1000"))
}

func TestParseCIDRsRejectsGarbage(t *testing.T) {
	nets := parseCIDRs([]string{"", "  ", "garbage"})
	assert.Empty(t, nets)

	nets = parseCIDRs([]string{"10.0.0.0/8", "127.0.0.1", "::1/128"})
	require.Len(t, nets, 3)
	assert.True(t, containsIP(nets, mustIP(t, "10.255.255.255")))
	assert.True(t, containsIP(nets, mustIP(t, "127.0.0.1")))
	assert.True(t, containsIP(nets, mustIP(t, "::1")))
	assert.False(t, containsIP(nets, mustIP(t, "11.0.0.1")))
}

func mustIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	require.NotNil(t, ip)
	return ip
}
