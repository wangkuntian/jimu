package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newMaskBufferLogger() (*zap.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(buf), zapcore.DebugLevel)
	return zap.New(WithMasking(core)), buf
}

func TestMaskFieldsRewritesSensitiveValues(t *testing.T) {
	fields := []zapcore.Field{
		zap.String("username", "alice"),
		zap.String("email", "alice@example.com"),
		zap.String("password", "s3cret"),
		zap.String("phone_number", "13812348888"),
		zap.Any("payload", map[string]any{"api_key": "k-1", "note": "keep"}),
	}
	masked := maskFields(fields)

	assert.Equal(t, "alice", masked[0].String)
	assert.Equal(t, "a***@example.com", masked[1].String)
	assert.Equal(t, "***", masked[2].String)
	assert.Equal(t, "138***8888", masked[3].String)
	payload := masked[4].Interface.(map[string]any)
	assert.Equal(t, "***", payload["api_key"])
	assert.Equal(t, "keep", payload["note"])
}

func TestMaskingCoreMasksLogOutput(t *testing.T) {
	l, buf := newMaskBufferLogger()
	l.Info("login",
		zap.String("username", "alice"),
		zap.String("email", "alice@example.com"),
		zap.String("password", "s3cret"),
	)

	out := buf.String()
	assert.Contains(t, out, "alice", "非敏感字段应保留")
	assert.Contains(t, out, "a***@example.com")
	assert.NotContains(t, out, "alice@example.com", "邮箱不得明文落日志")
	assert.NotContains(t, out, "s3cret", "密码不得明文落日志")
}

func TestMaskingCoreMasksWithFields(t *testing.T) {
	l, buf := newMaskBufferLogger()
	l.With(zap.String("phone", "13812348888")).Info("sms sent")

	out := buf.String()
	assert.Contains(t, out, "138***8888")
	assert.NotContains(t, out, "13812348888")
}
