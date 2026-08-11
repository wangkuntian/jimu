package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type validateStub struct {
	Name string `json:"name" binding:"required"`
}

func TestValidateJSONTranslatesFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		acceptLang string
		body       string
		wantMsg    string
	}{
		{"zh", "zh-CN,zh;q=0.8", `{}`, "Name 不能为空"},
		{"en", "en-US,en;q=0.9", `{}`, "Name is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Locale())
			r.POST("/", ValidateJSON(&validateStub{}), func(c *gin.Context) {
				c.String(200, "ok")
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", tt.acceptLang)
			r.ServeHTTP(w, req)
			if !strings.Contains(w.Body.String(), tt.wantMsg) {
				t.Fatalf("body = %s, want %q", w.Body.String(), tt.wantMsg)
			}
		})
	}
}
