package reporter

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewReporter_DisabledReturnsEmpty(t *testing.T) {
	r := NewReporter(ReporterConfig{Enabled: false}, nil)
	r.Report(context.Background(), errors.New("boom"), "k", "v")
	if !r.Flush(100 * time.Millisecond) {
		t.Fatal("Flush() should succeed for empty reporter")
	}
	// 不应 panic
}

func TestNewReporter_EnabledReturnsLogReporter(t *testing.T) {
	r := NewReporter(ReporterConfig{Enabled: true}, nil)
	if _, ok := r.(*logReporter); !ok {
		t.Fatalf("expected *logReporter, got %T", r)
	}
	r.Report(context.Background(), errors.New("boom"), "path", "/api/v1/users")
	if !r.Flush(100 * time.Millisecond) {
		t.Fatal("Flush() should succeed for log reporter")
	}
}

func TestLogReporter_CollectsFields(t *testing.T) {
	var gotMsg string
	var gotFields []interface{}
	r := &logReporter{logf: func(msg string, kvs ...interface{}) {
		gotMsg = msg
		gotFields = kvs
	}}
	r.Report(context.Background(), errors.New("boom"), "k1", "v1")
	if gotMsg != "error reported" {
		t.Fatalf("msg = %q", gotMsg)
	}
	if len(gotFields) != 4 || gotFields[0] != "error" || gotFields[3] != "v1" {
		t.Fatalf("fields = %v", gotFields)
	}
}

func TestLogReporter_IgnoresNilError(t *testing.T) {
	r := &logReporter{logf: func(string, ...interface{}) {}}
	r.Report(context.Background(), nil) // 不应 panic
}

func TestLogReporter_NilLogf(t *testing.T) {
	r := &logReporter{logf: nil}
	r.Report(context.Background(), errors.New("boom")) // 不应 panic
}

func TestEmptyReporter(t *testing.T) {
	var e emptyReporter
	e.Report(context.Background(), errors.New("x"))
	if !e.Flush(time.Second) {
		t.Fatal("emptyReporter.Flush() should succeed")
	}
}

func TestToPairs(t *testing.T) {
	p := toPairs([]string{"a", "1", "b", "2"})
	if len(p) != 4 || p[0] != "a" || p[3] != "2" {
		t.Fatalf("toPairs() = %v", p)
	}
	// 奇数长度容错
	p = toPairs([]string{"a", "1", "orphan"})
	if len(p) != 2 {
		t.Fatalf("toPairs() odd length = %v", p)
	}
}

func TestDefaultReporterConfig(t *testing.T) {
	cfg := DefaultReporterConfig()
	if !cfg.Enabled {
		t.Fatalf("default enabled = %v", cfg.Enabled)
	}
}
