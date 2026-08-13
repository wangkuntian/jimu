package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDUsesProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.String(200, c.GetString("request_id")) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "rid-123")
	r.ServeHTTP(w, req)
	assert.Equal(t, "rid-123", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "rid-123", w.Body.String())
}

func TestRequestIDGeneratesIfMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.String(200, c.GetString("request_id")) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, w.Header().Get("X-Request-ID"), w.Body.String())
}

func corsRouter(cfg CORSConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware(cfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORSAllowAllOrigins(t *testing.T) {
	r := corsRouter(CORSConfig{AllowedOrigins: []string{"*"}, MaxAge: 3600})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://any.example")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "3600", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORSAllowedOriginEcho(t *testing.T) {
	r := corsRouter(CORSConfig{AllowedOrigins: []string{"https://ok.example"}, MaxAge: 600})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://ok.example")
	r.ServeHTTP(w, req)
	assert.Equal(t, "https://ok.example", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Vary"), "Origin")
}

func TestCORSDisallowedOrigin(t *testing.T) {
	r := corsRouter(CORSConfig{AllowedOrigins: []string{"https://ok.example"}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://bad.example")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSPreflight(t *testing.T) {
	r := corsRouter(CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           100,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://x.example")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "GET,POST", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", w.Header().Get("Access-Control-Allow-Headers"))
}

func TestIsAllowedOrigin(t *testing.T) {
	allowed := []string{"a.com", "b.com"}
	assert.True(t, isAllowedOrigin("a.com", allowed))
	assert.True(t, isAllowedOrigin("b.com", allowed))
	assert.False(t, isAllowedOrigin("c.com", allowed))
	assert.False(t, isAllowedOrigin("", allowed))
	assert.False(t, isAllowedOrigin("a.com", nil))
}

func TestShouldLogBody(t *testing.T) {
	for _, ct := range []string{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain"} {
		assert.True(t, shouldLogBody(ct), "content-type=%s", ct)
	}
	for _, ct := range []string{"", "image/png", "multipart/form-data"} {
		assert.False(t, shouldLogBody(ct), "content-type=%s", ct)
	}
}

func TestDefaultLogConfig(t *testing.T) {
	cfg := DefaultLogConfig()
	assert.False(t, cfg.LogRequestBody)
	assert.False(t, cfg.LogResponseBody)
	assert.Equal(t, 1024, cfg.MaxBodyLogSize)
}

func TestLoggerMiddlewareRecordsRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New(config.LogConfig{Level: "error", Format: "console", Output: "stdout"})
	cfg := DefaultLogConfig()
	cfg.LogRequestBody = true
	cfg.LogResponseBody = true

	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(log, cfg))
	r.POST("/submit", func(c *gin.Context) { c.String(200, "response-data") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "response-data", w.Body.String())
}

func TestRecoveryCatchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":1005`)
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeadersFromConfig(config.SecurityConfig{}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
	// 空配置回填默认值
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src")
	assert.Contains(t, w.Header().Get("Referrer-Policy"), "strict-origin")
	assert.Contains(t, w.Header().Get("Permissions-Policy"), "camera")
}

func TestSecurityHeadersRespectsCustomValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.SecurityConfig{ContentTypeOptions: "custom-cto"}
	r := gin.New()
	r.Use(SecurityHeadersFromConfig(cfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, "custom-cto", w.Header().Get("X-Content-Type-Options"))
}

func TestSecurityAddVary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// 首次加入
	addVary(c, "Origin")
	assert.Equal(t, "Origin", c.Writer.Header().Get("Vary"))
	// 重复加入不生效
	addVary(c, "Origin")
	assert.Equal(t, "Origin", c.Writer.Header().Get("Vary"))
	// 追加第二个值
	addVary(c, "Accept-Encoding")
	assert.Equal(t, "Origin, Accept-Encoding", c.Writer.Header().Get("Vary"))
}

func TestSecurityHeadersNilValuesSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeadersFromConfig(config.SecurityConfig{
		ContentTypeOptions: "nosniff",
		// 其余字段留空，只验证写入的非空头
	}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
}
