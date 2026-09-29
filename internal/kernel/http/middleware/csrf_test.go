package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func csrfRouter(cfg CSRFConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF(cfg))
	r.GET("/view", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/submit", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func TestDefaultCSRFConfig(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("s"))
	assert.Equal(t, "X-CSR-Token", cfg.TokenHeader)
	assert.Equal(t, "_csrf", cfg.TokenField)
	assert.Equal(t, "csrf_token", cfg.CookieName)
	assert.Equal(t, 86400, cfg.CookieMaxAge)
	assert.Equal(t, []string{"GET", "HEAD", "OPTIONS"}, cfg.SafeMethods)
}

func TestCSRFGenerateTokenDeterministic(t *testing.T) {
	a := generateToken([]byte("secret"), 12345)
	b := generateToken([]byte("secret"), 12345)
	c := generateToken([]byte("other"), 12345)
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Len(t, a, 64)
}

func TestCSRFSafeMethodSetsCookie(t *testing.T) {
	r := csrfRouter(DefaultCSRFConfig([]byte("secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/view", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "csrf_token=")
}

func TestCSRFSafeMethodKeepsExistingCookie(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/view", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "existing"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))
}

func TestCSRFMissingCookie(t *testing.T) {
	r := csrfRouter(DefaultCSRFConfig([]byte("secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/submit", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"code":1003`)
}

func TestCSRFMissingRequestToken(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "tok"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFInvalidToken(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "cookie-token"})
	req.Header.Set(cfg.TokenHeader, "different")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFValidTokenFromHeader(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "same-token"})
	req.Header.Set(cfg.TokenHeader, "same-token")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCSRFValidTokenFromForm(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("_csrf=form-token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "form-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCSRFBearerTokenSkips(t *testing.T) {
	r := csrfRouter(DefaultCSRFConfig([]byte("secret")))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.Header.Set("Authorization", "Bearer abc.def")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCSRFSkipperSkips(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	cfg.Skipper = func(c *gin.Context) bool { return true }
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/submit", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCSRFSafeMethodsFallback(t *testing.T) {
	cfg := DefaultCSRFConfig([]byte("secret"))
	cfg.SafeMethods = nil
	r := csrfRouter(cfg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/view", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestIsSafeMethod(t *testing.T) {
	tests := []struct {
		method string
		safe   []string
		want   bool
	}{
		{"GET", []string{"GET", "HEAD", "OPTIONS"}, true},
		{"get", []string{"GET", "HEAD", "OPTIONS"}, true},
		{"POST", []string{"GET", "HEAD", "OPTIONS"}, false},
		{"HEAD", nil, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isSafeMethod(tt.method, tt.safe), "method=%s", tt.method)
	}
}
