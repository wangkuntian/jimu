package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appErrs "jimu/internal/shared/errors"
	"jimu/internal/shared/i18n"

	"github.com/gin-gonic/gin"
)

func TestFailTranslatesQuotaExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// locale 由 HTTP 层的 Locale 中间件写入上下文；测试直接注入，避免 response ↔ middleware 循环依赖
	tests := []struct {
		locale       string
		wantContains string
	}{
		{i18n.LangZH, "资源配额已用尽"},
		{i18n.LangEN, "quota exceeded"},
	}
	for _, tt := range tests {
		r := gin.New()
		r.GET("/", func(c *gin.Context) {
			c.Set("locale", tt.locale)
			Fail(c, appErrs.New(appErrs.CodeQuotaExceeded, "users quota exceeded"))
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
		if !strings.Contains(w.Body.String(), tt.wantContains) {
			t.Fatalf("body = %s, want contains %q", w.Body.String(), tt.wantContains)
		}
	}
}
