package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestOtelLogCore_ForwardsEntries 验证 zapcore.Core 桥接将条目写入 channel。
func TestOtelLogCore_ForwardsEntries(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{
		level: zapcore.DebugLevel,
		ch:    ch,
	}

	// Enabled/Check 过滤
	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello", Time: time.Now()}
	if !core.Enabled(ent.Level) {
		t.Fatal("info should be enabled at debug level")
	}

	ce := core.Check(ent, nil)
	if ce == nil {
		t.Fatal("Check() returned nil checked entry")
	}

	if err := core.Write(ent, []zapcore.Field{zap.String("k", "v")}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	select {
	case entry := <-ch:
		if entry.msg != "hello" || entry.level != zapcore.InfoLevel {
			t.Fatalf("unexpected entry: %+v", entry)
		}
		if len(entry.fields) != 1 {
			t.Fatalf("fields = %+v", entry.fields)
		}
	case <-time.After(time.Second):
		t.Fatal("entry not forwarded")
	}
}

// TestOtelLogCore_LevelFilter 验证级别过滤。
func TestOtelLogCore_LevelFilter(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{level: zapcore.WarnLevel, ch: ch}

	if core.Enabled(zapcore.InfoLevel) {
		t.Fatal("info should be disabled at warn level")
	}
	if !core.Enabled(zapcore.ErrorLevel) {
		t.Fatal("error should be enabled at warn level")
	}

	// 低于级别的条目应被 Check 拦截
	ce := core.Check(zapcore.Entry{Level: zapcore.InfoLevel}, nil)
	if ce != nil {
		t.Fatal("Check() should return nil for below-threshold level")
	}
}

// TestOtelLogCore_WithClonesFields 验证 With 返回带继承字段的新 core。
func TestOtelLogCore_WithClonesFields(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{level: zapcore.DebugLevel, ch: ch}

	cloned := core.With([]zapcore.Field{zap.String("svc", "jimu")})
	if core == cloned {
		t.Fatal("With() should return a new core")
	}

	_ = core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "a"}, nil)
	_ = cloned.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "b"}, nil)

	first := <-ch
	second := <-ch
	if len(first.fields) != 0 {
		t.Fatalf("original core should not inherit fields, got %+v", first.fields)
	}
	if len(second.fields) != 1 || second.fields[0].Key != "svc" {
		t.Fatalf("cloned core fields = %+v", second.fields)
	}
}

// TestOtelLogCore_BufferFullDrops 验证缓冲满时丢弃而非阻塞。
func TestOtelLogCore_BufferFullDrops(t *testing.T) {
	ch := make(chan logEntry, 1)
	core := &otelLogCore{level: zapcore.DebugLevel, ch: ch}

	// 填满缓冲
	_ = core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "full"}, nil)
	done := make(chan struct{})
	go func() {
		// 满时 Write 应立即返回（丢弃）
		for i := 0; i < 100; i++ {
			if err := core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "drop"}, nil); err != nil {
				t.Errorf("Write() error = %v", err)
				return
			}
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Write() blocked when buffer full")
	}
	<-ch // 清掉缓冲中的一条
}

// TestAttributeFromValue 验证字段类型保留（数值/布尔不退化字符串）。
func TestAttributeFromValue(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want attribute.Type
	}{
		{name: "string", in: "x", want: attribute.STRING},
		{name: "int", in: 7, want: attribute.INT64},
		{name: "int64", in: int64(7), want: attribute.INT64},
		{name: "float64", in: 1.5, want: attribute.FLOAT64},
		{name: "bool", in: true, want: attribute.BOOL},
		{name: "duration", in: 2 * time.Millisecond, want: attribute.INT64},
		{name: "time", in: time.Now(), want: attribute.STRING},
		{name: "string-slice", in: []string{"a"}, want: attribute.STRINGSLICE},
		{name: "bool-slice", in: []bool{true}, want: attribute.BOOLSLICE},
		{name: "int-slice", in: []int{1}, want: attribute.INT64SLICE},
		{name: "int64-slice", in: []int64{1}, want: attribute.INT64SLICE},
		{name: "float64-slice", in: []float64{1.5}, want: attribute.FLOAT64SLICE},
		{name: "fallback", in: struct{ A int }{A: 1}, want: attribute.STRING},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := attributeFromValue(c.name, c.in)
			require.Equal(t, c.want, got.Value.Type())
		})
	}
}

// TestTraceIDsFromFields 验证 trace_id/span_id 字符串字段解析为追踪标识。
func TestTraceIDsFromFields(t *testing.T) {
	tidHex := "0102030405060708090a0b0c0d0e0f10"
	sidHex := "0102030405060708"

	t.Run("valid", func(t *testing.T) {
		tid, sid, ok := traceIDsFromFields([]zapcore.Field{
			zap.String("trace_id", tidHex),
			zap.String("span_id", sidHex),
			zap.String("name", "x"),
		})
		require.True(t, ok)
		require.Equal(t, tidHex, tid.String())
		require.Equal(t, sidHex, sid.String())
	})

	t.Run("missing span id", func(t *testing.T) {
		_, _, ok := traceIDsFromFields([]zapcore.Field{
			zap.String("trace_id", tidHex),
		})
		require.False(t, ok)
	})

	t.Run("invalid hex", func(t *testing.T) {
		_, _, ok := traceIDsFromFields([]zapcore.Field{
			zap.String("trace_id", "zz"),
			zap.String("span_id", sidHex),
		})
		require.False(t, ok)
	})
}
