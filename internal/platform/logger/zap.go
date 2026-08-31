package logger

import (
	"context"
	"errors"
	"os"
	"syscall"

	"jimu/internal/config"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zap.SugaredLogger
	level *zap.AtomicLevel
}

func (l *Logger) Sync() error {
	err := l.SugaredLogger.Sync()
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTTY) {
		return nil
	}
	return err
}

// SetLevel 动态调整日志级别（支持运行热更新）
func (l *Logger) SetLevel(level string) error {
	var lv zapcore.Level
	switch level {
	case "debug":
		lv = zapcore.DebugLevel
	case "info":
		lv = zapcore.InfoLevel
	case "warn":
		lv = zapcore.WarnLevel
	case "error":
		lv = zapcore.ErrorLevel
	default:
		return errors.New("invalid log level: " + level)
	}
	l.level.SetLevel(lv)
	return nil
}

// WithContext 从 context 中提取 trace_id 和 span_id 加入日志字段
func (l *Logger) WithContext(ctx context.Context) *Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		withTrace := l.With(
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		)
		return &Logger{withTrace, l.level}
	}
	return l
}

// New 创建 Logger。extraCores 为附加输出（如 OpenObserve OTLP 日志通道），
// 与本地输出并行写入；附加 core 的失败不影响主链路。
func New(cfg config.LogConfig, extraCores ...zapcore.Core) *Logger {
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "msg",
		CallerKey:    "caller",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.Output == "stdout" || cfg.Output == "" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		writeSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
	}

	atomicLevel := zap.NewAtomicLevelAt(level)
	core := zapcore.NewCore(encoder, writeSyncer, atomicLevel)
	if len(extraCores) > 0 {
		cores := make([]zapcore.Core, 0, 1+len(extraCores))
		cores = append(cores, core)
		cores = append(cores, extraCores...)
		core = zapcore.NewTee(cores...)
	}
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{zapLogger.Sugar(), &atomicLevel}
}
