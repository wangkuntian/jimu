package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	apikey "jimu/internal/capabilities/apikey"
	"jimu/internal/kernel/tenant"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// tenantHeader 用请求头模拟租户注入（真实链路由 tenant.Middleware 从 JWT tid 注入）
const tenantHeader = "X-Test-Tenant"

func tenantInjectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.GetHeader(tenantHeader); v != "" {
			if tid, err := strconv.ParseUint(v, 10, 64); err == nil && tid != 0 {
				c.Request = c.Request.WithContext(tenant.WithTenant(c.Request.Context(), tid))
			}
		}
		c.Next()
	}
}

func TestTenantRateLimitMiddleware(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(tenantInjectMiddleware())
	r.Use(TenantRateLimitMiddleware(rdb, 2, time.Minute))
	r.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(tid string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		if tid != "" {
			req.Header.Set(tenantHeader, tid)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 平台级视角（无租户）不限制
	for i := 0; i < 5; i++ {
		assert.Equal(t, http.StatusOK, do(""))
	}

	// 租户 1：窗口内 2 次后 429
	assert.Equal(t, http.StatusOK, do("1"))
	assert.Equal(t, http.StatusOK, do("1"))
	assert.Equal(t, http.StatusTooManyRequests, do("1"))

	// 租户 2 独立计数
	assert.Equal(t, http.StatusOK, do("2"))
}

func TestAPIKeyRateLimitMiddleware(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if v := c.GetHeader("X-Test-KeyID"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 64); err == nil {
				req := c.Request
				c.Request = req.WithContext(apikey.ContextWithAPIKey(req.Context(), &apikey.APIKey{ID: id}))
			}
		}
		c.Next()
	})
	r.Use(APIKeyRateLimitMiddleware(rdb, 2, time.Minute))
	r.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(keyID string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		if keyID != "" {
			req.Header.Set("X-Test-KeyID", keyID)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 未携带 API Key 的请求跳过该维度
	for i := 0; i < 5; i++ {
		assert.Equal(t, http.StatusOK, do(""))
	}

	// Key 1：窗口内 2 次后 429
	assert.Equal(t, http.StatusOK, do("1"))
	assert.Equal(t, http.StatusOK, do("1"))
	assert.Equal(t, http.StatusTooManyRequests, do("1"))

	// Key 2 独立计数
	assert.Equal(t, http.StatusOK, do("2"))
}
