package observability

import (
	"context"
	"testing"
)

func TestInitTracingDisabledReturnsNoOp(t *testing.T) {
	tp, err := InitTracing(context.Background(), TracingConfig{Enabled: false})
	if err != nil {
		t.Fatalf("InitTracing disabled err = %v, want nil", err)
	}
	if tp == nil {
		t.Fatal("InitTracing disabled tp = nil, want non-nil")
	}
	// 禁用时无需 shutdown，但应不报错
	if err := ShutdownTracing(context.Background(), tp); err != nil {
		t.Fatalf("ShutdownTracing err = %v, want nil", err)
	}
}

func TestShutdownTracingNil(t *testing.T) {
	if err := ShutdownTracing(context.Background(), nil); err != nil {
		t.Fatalf("ShutdownTracing(nil) err = %v, want nil", err)
	}
}

func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()
	if cfg.Enabled {
		t.Fatal("DefaultTracingConfig.Enabled = true, want false")
	}
	if cfg.ServiceName == "" {
		t.Fatal("DefaultTracingConfig.ServiceName empty")
	}
	if cfg.SampleRate <= 0 {
		t.Fatalf("DefaultTracingConfig.SampleRate = %v, want > 0", cfg.SampleRate)
	}
}

func TestTraceFromContextDisabled(t *testing.T) {
	// 禁用状态下（未设置 propagator）应返回空串，不 panic
	tp, _ := InitTracing(context.Background(), TracingConfig{Enabled: false})
	defer tp.Shutdown(context.Background())

	parent, state := TraceFromContext(context.Background())
	if parent != "" || state != "" {
		t.Fatalf("TraceFromContext = (%q, %q), want empty when disabled", parent, state)
	}
}

type ctxKey string

func TestContextWithTraceEmptyPassthrough(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	got := ContextWithTrace(ctx, "", "")
	if got != ctx {
		t.Fatal("ContextWithTrace empty should return original ctx")
	}
}
