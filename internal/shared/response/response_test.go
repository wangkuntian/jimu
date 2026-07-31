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
	if !strings.Contains(body, "request-123") || !strings.Contains(body, "internal error") {
		t.Fatalf("response missing safe diagnostics: %s", body)
	}
}
