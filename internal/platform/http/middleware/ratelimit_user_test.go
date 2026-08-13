package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyFuncPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 优先 user_id
	c1, _ := gin.CreateTestContext(httptest.NewRecorder())
	c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c1.Set("user_id", uint64(42))
	assert.Equal(t, "user:42", defaultKeyFunc(c1))

	// 其次 API Key
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c2.Request.Header.Set("X-Api-Key", "abc")
	assert.Equal(t, "apikey:abc", defaultKeyFunc(c2))

	// 最后 IP
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c3.Request.RemoteAddr = "9.9.9.9:1"
	assert.Equal(t, "ip:9.9.9.9", defaultKeyFunc(c3))
}

func TestUserRateLimiterOptions(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	r := NewUserRateLimiter(rdb, 5, time.Minute,
		WithKeyPrefix("rl:custom"),
		WithKeyFunc(func(c *gin.Context) string { return "fixed" }),
	)
	assert.Equal(t, "rl:custom", r.prefix)
	assert.Equal(t, "fixed", r.keyFunc(nil))
}

func userRateRouter(r *UserRateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.Use(r.Middleware())
	eng.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })
	return eng
}

func TestUserRateLimiterAllowsWithinLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	r := NewUserRateLimiter(rdb, 2, time.Minute)
	eng := userRateRouter(r)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "203.0.113.9:1"
		eng.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "请求 %d 应放行", i)
		assert.Equal(t, "2", w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	}
}

func TestUserRateLimiterRejectsOverLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	r := NewUserRateLimiter(rdb, 2, time.Minute)
	eng := userRateRouter(r)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "203.0.113.9:1"
		eng.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.RemoteAddr = "203.0.113.9:1"
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), `"code":1007`)
}

func TestUserRateLimiterFailOpenOnRedisError(t *testing.T) {
	// 指向不可达地址，Pipeline Exec 出错 → fail-open 放行
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	defer rdb.Close()

	r := NewUserRateLimiter(rdb, 2, time.Minute)
	eng := userRateRouter(r)

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
