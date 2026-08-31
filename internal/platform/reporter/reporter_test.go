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

func TestNewReporter_NoDSNReturnsLogOnly(t *testing.T) {
	r := NewReporter(ReporterConfig{Enabled: true, DSN: ""}, nil)
	r.Report(context.Background(), errors.New("boom"), "path", "/api/v1/users")
	if !r.Flush(100 * time.Millisecond) {
		t.Fatal("Flush() should succeed for log reporter")
	}
}

func TestNewReporter_InvalidDSNFallsBackToLog(t *testing.T) {
	// DSN 格式非法时回退日志实现，不阻断启动
	r := NewReporter(ReporterConfig{Enabled: true, DSN: "not-a-dsn"}, nil)
	r.Report(context.Background(), errors.New("boom"))
	if !r.Flush(100 * time.Millisecond) {
		t.Fatal("Flush() should succeed after fallback")
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

func TestDualReporter(t *testing.T) {
	inner := &logReporter{logf: func(string, ...interface{}) {}}
	other := &logReporter{logf: func(string, ...interface{}) {}}
	d := &dualReporter{log: inner, sentry: other}
	d.Report(context.Background(), errors.New("boom"), "a", "b")
	if !d.Flush(100 * time.Millisecond) {
		t.Fatal("dualReporter.Flush() should succeed")
	}
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
	if cfg.SampleRate != 1.0 {
		t.Fatalf("default sample rate = %v", cfg.SampleRate)
	}
}
