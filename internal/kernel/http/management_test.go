package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jimu/internal/kernel/health"
)

type passingChecker struct{}

func (passingChecker) Ping(context.Context) error { return nil }

func TestManagementRouterExposure(t *testing.T) {
	readiness := health.NewReadiness(time.Second, passingChecker{})

	tests := []struct {
		path        string
		enablePprof bool
		withMetrics bool
		want        int
	}{
		{"/livez", false, false, stdhttp.StatusOK},
		{"/readyz", false, false, stdhttp.StatusOK},
		{"/metrics", false, true, stdhttp.StatusOK},
		// 未注入 metrics handler 时不暴露 /metrics：内核不再硬绑 Prometheus。
		{"/metrics", false, false, stdhttp.StatusNotFound},
		{"/debug/pprof/", false, false, stdhttp.StatusNotFound},
		{"/debug/pprof/", true, false, stdhttp.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var metrics MetricsHandler
			if tt.withMetrics {
				metrics = func() stdhttp.Handler {
					return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
						w.WriteHeader(stdhttp.StatusOK)
					})
				}
			}
			w := httptest.NewRecorder()
			HealthRouter(readiness, metrics, tt.enablePprof).ServeHTTP(
				w,
				httptest.NewRequest(stdhttp.MethodGet, tt.path, nil),
			)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
