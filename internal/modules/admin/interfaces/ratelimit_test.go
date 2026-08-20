package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jimu/internal/platform/auth"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRateLimitHandler(t *testing.T) (*AdminRateLimitHandler, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewAdminRateLimitHandler(client), client
}

func TestAdminRateLimitAuthPeek_MissingParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newRateLimitHandler(t)
	r := gin.New()
	r.GET("/peek", handler.AuthPeek)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/peek?scope=login", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// response.Fail 只回翻译后的通用消息，用错误码 1001(CodeInvalidParam) 判定参数错误
	assert.Contains(t, w.Body.String(), `"code":1001`)
}

func TestAdminRateLimitAuthPeek_KeyAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newRateLimitHandler(t)
	r := gin.New()
	r.GET("/peek", handler.AuthPeek)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/peek?scope=login&key=ip:1.2.3.4", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Code int `json:"code"`
		Data struct {
			Count int64  `json:"count"`
			TTLms int64  `json:"ttl_ms"`
			Scope string `json:"scope"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, int64(0), body.Data.Count, "不存在 key 计数应为 0")
	assert.Equal(t, int64(0), body.Data.TTLms, "不存在 key TTL 应为 0")
	assert.Equal(t, "login", body.Data.Scope)
}

// TestAdminRateLimitAuthPeek_ExistingCount 用真实 Limiter 写入计数后验证 peek 读出一致
func TestAdminRateLimitAuthPeek_ExistingCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, client := newRateLimitHandler(t)
	ctx := context.Background()

	// 用真实 Limiter 触发 3 次登录限流（limit=5, window=1m）
	limiter := auth.NewLimiter(client, true)
	for i := 0; i < 3; i++ {
		ok, err := limiter.Allow(ctx, "login", "ip:10.0.0.1", 5, time.Minute)
		require.NoError(t, err)
		require.True(t, ok)
	}

	r := gin.New()
	r.GET("/peek", handler.AuthPeek)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/peek?scope=login&key=ip:10.0.0.1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data struct {
			Count    int64  `json:"count"`
			TTLms    int64  `json:"ttl_ms"`
			RedisKey string `json:"redis_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, int64(3), body.Data.Count, "peek 应读到已消费的 3 次计数")
	assert.Greater(t, body.Data.TTLms, int64(0), "存在 key 应有剩余 TTL")
	assert.Equal(t, auth.LimitKey("login", "ip:10.0.0.1"), body.Data.RedisKey)
}
