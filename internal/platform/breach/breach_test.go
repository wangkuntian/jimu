package breach

import (
	"context"
	"crypto/sha1" //nolint:gosec // 测试需要按协议计算期望指纹
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/platform/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sha1Upper 返回口令的大写 SHA-1 十六进制
func sha1Upper(password string) string {
	//nolint:gosec // 与实现一致：HIBP 协议指纹
	sum := sha1.Sum([]byte(password))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// recorded 记录最后一次请求，供断言 k-匿名与请求头
type recorded struct {
	path   string
	header http.Header
}

func newTestChecker(t *testing.T, handler http.HandlerFunc) (Checker, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.path = r.URL.Path
		rec.header = r.Header.Clone()
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	c := &hibpChecker{client: httpclient.New(httpclient.Config{TimeoutSec: 5}), baseURL: srv.URL + "/range/"}
	return c, rec
}

func TestIsBreachedOnlySendsHashPrefix(t *testing.T) {
	hash := sha1Upper("password123")
	var body string
	checker, rec := newTestChecker(t, func(w http.ResponseWriter, _ *http.Request) {
		// 前缀 + 无关后缀：不应命中
		fmt.Fprintf(w, "0000000000000000000000000000000000:1\n")
		// 命中行：大小写不一致也应识别
		fmt.Fprintf(w, "%s:42\n", strings.ToLower(hash[prefixLen:]))
		body = "ok"
	})

	breached, err := checker.IsBreached(context.Background(), "password123")
	require.NoError(t, err)
	assert.True(t, breached)
	assert.Equal(t, hash[:prefixLen], strings.TrimPrefix(rec.path, "/range/"), "只发送 5 位前缀")
	assert.NotContains(t, rec.path, hash[prefixLen:], "完整哈希不出网")
	assert.Equal(t, "true", rec.header.Get("Add-Padding"))
	assert.Equal(t, userAgent, rec.header.Get("User-Agent"))
	assert.Equal(t, "ok", body)
}

func TestIsBreachedPaddingAndMiss(t *testing.T) {
	checker, _ := newTestChecker(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "0000000000000000000000000000000000:3\n")
		fmt.Fprint(w, "ffffffffffffffffffffffffffffffffff:9\n")
		// HIBP 的填充行没有计数
		fmt.Fprint(w, "1111111111111111111111111111111111\n")
	})

	breached, err := checker.IsBreached(context.Background(), "a-very-unique-passphrase")
	require.NoError(t, err)
	assert.False(t, breached, "未命中任何后缀应放行")
}

func TestIsBreachedErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }},
		{"bad status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, _ := newTestChecker(t, tt.handler)
			breached, err := checker.IsBreached(context.Background(), "password123")
			assert.Error(t, err)
			assert.False(t, breached, "出错时不得误判为已泄露")
		})
	}
}

func TestIsBreachedHonoursContext(t *testing.T) {
	checker, _ := newTestChecker(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, "0000000000000000000000000000000000:1\n")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	breached, err := checker.IsBreached(ctx, "password123")
	assert.Error(t, err)
	assert.False(t, breached)
}

func TestNewUsesOfficialEndpoint(t *testing.T) {
	c := New(httpclient.New(httpclient.Config{}))
	impl, ok := c.(*hibpChecker)
	require.True(t, ok)
	assert.Equal(t, rangeURL, impl.baseURL)
}
