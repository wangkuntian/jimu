package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appErrs "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

func TestFailMapsApplicationCodeToHTTPStatus(t *testing.T) {
	tests := []struct {
		code int
		want int
	}{
		{appErrs.CodeInvalidParam, http.StatusBadRequest},
		{appErrs.CodeUnauthorized, http.StatusUnauthorized},
		{appErrs.CodeForbidden, http.StatusForbidden},
		{appErrs.CodeNotFound, http.StatusNotFound},
		{appErrs.CodeRateLimited, http.StatusTooManyRequests},
		{appErrs.CodeTimeout, http.StatusGatewayTimeout},
		{appErrs.CodeInternalError, http.StatusInternalServerError},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(http.StatusText(tt.want), func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) { Fail(c, appErrs.New(tt.code, "public message")) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestFailHidesInternalCauseAndIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "request-123")
		Fail(c, appErrs.Wrap(appErrs.CodeInternalError, "database failed", errors.New("secret DSN")))
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "database failed") || strings.Contains(body, "secret DSN") {
		t.Fatalf("response leaked internal error: %s", body)
	}
	if !strings.Contains(body, "request-123") || !strings.Contains(body, "服务器内部错误") {
		t.Fatalf("response missing safe diagnostics: %s", body)
	}
}

func TestFailTranslatesByLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{"zh", "zh", "服务器内部错误"},
		{"en", "en", "internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				c.Set("locale", tt.locale)
				Fail(c, appErrs.New(appErrs.CodeInternalError, "db down"))
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if !strings.Contains(w.Body.String(), tt.want) {
				t.Fatalf("body = %s, want %q", w.Body.String(), tt.want)
			}
		})
	}
}

func TestCreatedUsesStandardEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/", func(c *gin.Context) {
		c.Set("request_id", "rid-created")
		Created(c, gin.H{"id": uint64(7)})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if !strings.Contains(w.Body.String(), `"request_id":"rid-created"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestNoContentWritesNoBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/", NoContent)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}
