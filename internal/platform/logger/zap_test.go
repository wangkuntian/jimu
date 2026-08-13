package logger

import (
	"context"
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewConsoleStdout(t *testing.T) {
	l := New(config.LogConfig{Level: "debug", Format: "console", Output: "stdout"})
	require.NotNil(t, l)
	l.Info("hello")
	_ = l.Sync() // stdout 在测试环境可能不可 sync，忽略错误
}

func TestNewJSONLevels(t *testing.T) {
	for _, lv := range []string{"debug", "info", "warn", "error"} {
		l := New(config.LogConfig{Level: lv, Format: "json", Output: "stdout"})
		assert.NotNil(t, l)
	}
	// 非法级别回退 info
	l := New(config.LogConfig{Level: "bogus", Format: "json", Output: "stdout"})
	assert.NotNil(t, l)
}

func TestSetLevel(t *testing.T) {
	l := New(config.LogConfig{Level: "info", Output: "stdout"})
	require.NoError(t, l.SetLevel("debug"))
	require.NoError(t, l.SetLevel("warn"))
	require.NoError(t, l.SetLevel("error"))
	err := l.SetLevel("bogus")
	assert.Error(t, err)
}

func TestWithContext(t *testing.T) {
	l := New(config.LogConfig{Level: "info", Output: "stdout"})

	// 无 span 返回原 logger
	assert.Same(t, l, l.WithContext(context.Background()))

	// 有有效 span 返回带 trace 字段的 logger
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	withCtx := l.WithContext(ctx)
	assert.NotSame(t, l, withCtx)
	withCtx.Info("with trace")
	_ = withCtx.Sync()
}

func TestFileOutput(t *testing.T) {
	dir := t.TempDir()
	l := New(config.LogConfig{
		Level:   "info",
		Format:  "json",
		Output:  dir + "/app.log",
		MaxSize: 1,
		MaxAge:  1,
	})
	l.Info("to file")
	require.NoError(t, l.Sync())
}
