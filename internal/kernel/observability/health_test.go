package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type checkerFunc func(context.Context) error

func (f checkerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func TestReadinessStatus(t *testing.T) {
	tests := []struct {
		name     string
		dbErr    error
		redisErr error
		want     int
	}{
		{"ready", nil, nil, http.StatusOK},
		{"database down", errors.New("down"), nil, http.StatusServiceUnavailable},
		{"redis down", nil, errors.New("down"), http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterHealth(mux, NewReadiness(50*time.Millisecond,
				checkerFunc(func(context.Context) error { return tt.dbErr }),
				checkerFunc(func(context.Context) error { return tt.redisErr }),
			))

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestReadinessBoundsCheckerDuration(t *testing.T) {
	readiness := NewReadiness(10*time.Millisecond, checkerFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}))
	started := time.Now()
	if readiness.Ready(context.Background()) {
		t.Fatal("Ready() = true, want false")
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("readiness check took %s", elapsed)
	}
}
