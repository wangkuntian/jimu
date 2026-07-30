package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jimu/internal/platform/observability"
)

type passingChecker struct{}

func (passingChecker) Ping(context.Context) error { return nil }

func TestManagementRouterExposure(t *testing.T) {
	readiness := observability.NewReadiness(time.Second, passingChecker{})

	tests := []struct {
		path        string
		enablePprof bool
		want        int
	}{
		{"/livez", false, stdhttp.StatusOK},
		{"/readyz", false, stdhttp.StatusOK},
		{"/metrics", false, stdhttp.StatusOK},
		{"/debug/pprof/", false, stdhttp.StatusNotFound},
		{"/debug/pprof/", true, stdhttp.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			HealthRouter(readiness, tt.enablePprof).ServeHTTP(
				w,
				httptest.NewRequest(stdhttp.MethodGet, tt.path, nil),
			)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
