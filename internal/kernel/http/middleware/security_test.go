package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/config"

	"github.com/gin-gonic/gin"
)

func securityRouter(cfg config.HTTPConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Security(cfg))
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/resource", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func TestSecurityRejectsOversizedBody(t *testing.T) {
	r := securityRouter(config.HTTPConfig{MaxBodyBytes: 4})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("12345")))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestSecurityUsesOriginAllowList(t *testing.T) {
	r := securityRouter(config.HTTPConfig{
		MaxBodyBytes:   1024,
		AllowedOrigins: []string{"https://allowed.example"},
	})

	for _, tt := range []struct {
		origin string
		want   string
	}{
		{"https://allowed.example", "https://allowed.example"},
		{"https://denied.example", ""},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		req.Header.Set("Origin", tt.origin)
		r.ServeHTTP(w, req)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
			t.Fatalf("origin %q: header = %q, want %q", tt.origin, got, tt.want)
		}
	}
}

func TestSecurityHandlesAllowedPreflight(t *testing.T) {
	r := securityRouter(config.HTTPConfig{
		MaxBodyBytes:   1024,
		AllowedOrigins: []string{"https://allowed.example"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "https://allowed.example")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("Vary = %q", got)
	}
}
