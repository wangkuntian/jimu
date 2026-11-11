package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func ratelimitRouter(rl *RateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func ratelimitRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "203.0.113.7:1234"
	return req
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 1)
	require.NotNil(t, rl)
	assert.Equal(t, rate.Limit(1), rl.rate)
	assert.Equal(t, 1, rl.burst)
}

func TestRateLimiterSameIPReusesLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1000), 10)
	l1 := rl.getLimiter("1.2.3.4")
	l2 := rl.getLimiter("1.2.3.4")
	l3 := rl.getLimiter("5.6.7.8")
	assert.Same(t, l1, l2)
	assert.NotSame(t, l1, l3)
}

func TestRateLimiterAllowsThenRejects(t *testing.T) {
	// 慢速率：消耗令牌后短时间内不会回填，拒绝分支确定性触发
	rl := NewRateLimiter(rate.Limit(0.1), 2)
	r := ratelimitRouter(rl)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, ratelimitRequest("/resource"))
		assert.Equal(t, http.StatusOK, w.Code, "request %d 应放行", i)
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Limit"))
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, ratelimitRequest("/resource"))
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第三个请求应被限流")
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), `"code":1007`)
}

func TestRateLimiterCleanupLocked(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1000), 5)
	// 满桶（未消费）条目应被清理
	rl.visitors["idle1"] = rate.NewLimiter(rate.Limit(1000), 5)
	rl.visitors["idle2"] = rate.NewLimiter(rate.Limit(1000), 5)
	// 已消费条目保留
	used := rate.NewLimiter(rate.Limit(1000), 5)
	require.True(t, used.Allow())
	rl.visitors["active"] = used

	rl.cleanupLocked()

	_, ok1 := rl.visitors["idle1"]
	_, ok2 := rl.visitors["idle2"]
	_, okActive := rl.visitors["active"]
	assert.False(t, ok1, "满桶 idle1 应被清理")
	assert.False(t, ok2, "满桶 idle2 应被清理")
	assert.True(t, okActive, "已消费 active 应保留")
}

func TestGlobalRateLimitDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GlobalRateLimit(0, 0))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := ratelimitRequest("/")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
}

func TestGlobalRateLimitRejectsBurstOverflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GlobalRateLimit(1, 1)) // 1 req/s，桶容量 1
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, ratelimitRequest("/"))
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, ratelimitRequest("/"))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code, "桶耗尽后应限流")
}
