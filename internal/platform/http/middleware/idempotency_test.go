package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIDKey = "aaaaaaaa-1111-2222-3333-444444444444"

func idempotencyRouter(rdb *redis.Client) (*gin.Engine, *atomic.Int32) {
	gin.SetMode(gin.TestMode)
	var calls atomic.Int32
	r := gin.New()
	r.Use(IdempotencyMiddleware(rdb, time.Minute))
	r.POST("/pay", func(c *gin.Context) {
		calls.Add(1)
		c.JSON(http.StatusOK, gin.H{"result": "ok"})
	})
	return r, &calls
}

func newRedisForTest(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestIdempotencySkipsWithoutKey(t *testing.T) {
	rdb := newRedisForTest(t)
	r, calls := idempotencyRouter(rdb)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/pay", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(1), calls.Load())
}

func TestIdempotencyRejectsInvalidKey(t *testing.T) {
	rdb := newRedisForTest(t)
	r, _ := idempotencyRouter(rdb)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req.Header.Set("Idempotency-Key", "short")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req2.Header.Set("Idempotency-Key", string(bytes.Repeat([]byte("k"), 129)))
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestIdempotencyCachesAndReplays(t *testing.T) {
	rdb := newRedisForTest(t)
	r, calls := idempotencyRouter(rdb)

	// 第一次：执行 handler 并缓存
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req1.Header.Set("Idempotency-Key", testIDKey)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, int32(1), calls.Load())

	// 第二次：命中缓存，handler 不执行
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req2.Header.Set("Idempotency-Key", testIDKey)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, w1.Body.String(), w2.Body.String())
	assert.Equal(t, "application/json; charset=utf-8", w2.Header().Get("Content-Type"))
}

func TestIdempotencyDoesNotCacheFailure(t *testing.T) {
	rdb := newRedisForTest(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IdempotencyMiddleware(rdb, time.Minute))
	r.POST("/fail", func(c *gin.Context) { c.JSON(http.StatusInternalServerError, gin.H{"e": "x"}) })

	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	req.Header.Set("Idempotency-Key", "bbbbbbbb-1111-2222-3333-444444444444")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIdempotencyCachedResponseSerializes(t *testing.T) {
	rdb := newRedisForTest(t)
	r, _ := idempotencyRouter(rdb)

	req := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req.Header.Set("Idempotency-Key", testIDKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// redis 中确有缓存且结构可反序列化
	keys, err := rdb.Keys(req.Context(), "idempotency:*").Result()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	raw, err := rdb.Get(req.Context(), keys[0]).Result()
	require.NoError(t, err)
	var cached cachedResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &cached))
	assert.Equal(t, http.StatusOK, cached.Status)
	assert.NotEmpty(t, cached.Body)
}
