package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocaleParsesAcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"en", "en-US,en;q=0.9", "en"},
		{"zh", "zh-CN,zh;q=0.8", "zh"},
		{"missing defaults zh", "", "zh"},
		{"unsupported falls back zh", "fr-FR", "zh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Locale())
			r.GET("/", func(c *gin.Context) {
				c.String(200, c.GetString("locale"))
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Accept-Language", tt.header)
			}
			r.ServeHTTP(w, req)
			if w.Body.String() != tt.want {
				t.Fatalf("locale = %q, want %q", w.Body.String(), tt.want)
			}
		})
	}
}
