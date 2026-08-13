package db

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"jimu/internal/platform/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newBufferLogger 构建写入内存 buffer 的 zap logger（level 字段留空即可，WithContext 不依赖它）
func newBufferLogger(t *testing.T) (*logger.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	cfg := zapcore.EncoderConfig{
		TimeKey:    "ts",
		LevelKey:   "level",
		MessageKey: "msg",
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(&buf), zap.NewAtomicLevelAt(zapcore.DebugLevel))
	sugar := zap.New(core).Sugar()
	return &logger.Logger{SugaredLogger: sugar}, &buf
}

func newGormLogger(t *testing.T) (*gormLogger, *bytes.Buffer) {
	t.Helper()
	log, buf := newBufferLogger(t)
	return NewGormLogger(log, SlowQueryThreshold).(*gormLogger), buf
}

func TestNewGormLogger_DefaultThreshold(t *testing.T) {
	l := NewGormLogger(nil, 0)
	gl, ok := l.(*gormLogger)
	require.True(t, ok)
	assert.Equal(t, SlowQueryThreshold, gl.threshold)
}

func TestNewGormLogger_CustomThreshold(t *testing.T) {
	l := NewGormLogger(nil, time.Second)
	gl, ok := l.(*gormLogger)
	require.True(t, ok)
	assert.Equal(t, time.Second, gl.threshold)
}

func TestGormLogger_LogMode(t *testing.T) {
	l, _ := newGormLogger(t)
	assert.Same(t, l, l.LogMode(gormlogger.Info))
}

func TestGormLogger_InfoRedactsSensitive(t *testing.T) {
	l, buf := newGormLogger(t)
	l.Info(context.Background(), "user info", "password", "secret123", "username", "alice")
	out := buf.String()
	assert.Contains(t, out, "user info")
	assert.Contains(t, out, "alice")
	assert.Contains(t, out, "***")
	assert.NotContains(t, out, "secret123")
}

func TestGormLogger_WarnRedactsSensitive(t *testing.T) {
	l, buf := newGormLogger(t)
	l.Warn(context.Background(), "slow", "access_token", "tok123")
	out := buf.String()
	assert.Contains(t, out, "***")
	assert.NotContains(t, out, "tok123")
}

func TestGormLogger_ErrorRedactsSensitive(t *testing.T) {
	l, buf := newGormLogger(t)
	l.Error(context.Background(), "oops", "phone", "13800000000")
	out := buf.String()
	assert.Contains(t, out, "***")
	assert.NotContains(t, out, "13800000000")
}

func TestGormLogger_TraceError(t *testing.T) {
	l, buf := newGormLogger(t)
	begin := time.Now()
	l.Trace(context.Background(), begin, func() (string, int64) { return "SELECT 1", 3 }, errors.New("boom"))
	out := buf.String()
	assert.Contains(t, out, "\"error\":\"boom\"")
}

func TestGormLogger_TraceRecordNotFoundIsSilent(t *testing.T) {
	l, buf := newGormLogger(t)
	begin := time.Now()
	l.Trace(context.Background(), begin, func() (string, int64) { return "SELECT * FROM users", 0 }, gorm.ErrRecordNotFound)
	assert.Empty(t, buf.String())
}

func TestGormLogger_TraceSlowQueryWarns(t *testing.T) {
	l, buf := newGormLogger(t)
	begin := time.Now().Add(-SlowQueryThreshold - time.Millisecond)
	l.Trace(context.Background(), begin, func() (string, int64) { return "SELECT 1", 1 }, nil)
	out := buf.String()
	assert.Contains(t, out, "slow query detected")
	assert.Contains(t, out, "SELECT 1")
}

func TestGormLogger_TraceFastQuerySilent(t *testing.T) {
	l, buf := newGormLogger(t)
	begin := time.Now()
	l.Trace(context.Background(), begin, func() (string, int64) { return "SELECT 1", 1 }, nil)
	assert.Empty(t, buf.String())
}

func TestSanitizeArgs(t *testing.T) {
	args := []interface{}{"password", "secret", "name", "bob", "idcard", "110101", 3}
	out := sanitizeArgs(args)
	assert.Equal(t, "***", out[1])
	assert.Equal(t, "bob", out[3])
	assert.Equal(t, "***", out[5])
	assert.Equal(t, 3, out[6])
}

func TestSanitizeSQL(t *testing.T) {
	// 非 insert/update 原样返回
	sel := "SELECT password FROM users"
	assert.Equal(t, sel, sanitizeSQL(sel))

	// insert 超长截断
	longInsert := "INSERT INTO users VALUES (" + strings.Repeat("x", 300) + ")"
	got := sanitizeSQL(longInsert)
	assert.Equal(t, 200, len(strings.SplitN(got, "...[truncated]", 2)[0]))
	assert.True(t, strings.HasSuffix(got, "...[truncated]"))

	// 短 insert 原样
	short := "INSERT INTO users VALUES (1)"
	assert.Equal(t, short, sanitizeSQL(short))

	// update 超长截断
	longUpdate := "UPDATE users SET note = '" + strings.Repeat("y", 300) + "'"
	assert.True(t, strings.HasSuffix(sanitizeSQL(longUpdate), "...[truncated]"))
}

func TestIsSensitiveField(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"password", true},
		{"PASSWORD", true},
		{"refresh_token", true},
		{"credit_card_number", true},
		{"mobile_phone", true},
		{"username", false},
		{"email", false},
		{"created_at", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, isSensitiveField(c.name), c.name)
	}
}
