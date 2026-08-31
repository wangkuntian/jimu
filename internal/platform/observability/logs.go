package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otlploggrpc "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelog "go.opentelemetry.io/otel/log"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

// LogExporter 通过 OTLP/gRPC 将结构化日志推送到 OpenObserve。
// 通过 ZapCore 桥接接入现有日志链路：所有写出的日志条目异步批量发送，
// 发送失败静默丢弃（本地文件/stdout 输出不受影响）。
type LogExporter struct {
	provider *logsdk.LoggerProvider
	logger   otelog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLogExporter 创建 OTLP logs exporter。
func NewLogExporter(ctx context.Context, cfg TracingConfig) (*LogExporter, error) {
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.Endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp logs exporter: %w", err)
	}

	provider := logsdk.NewLoggerProvider(logsdk.WithProcessor(logsdk.NewBatchProcessor(exporter)))
	logger := provider.Logger(defaulted(cfg.ServiceName, "jimu"))
	runCtx, cancel := context.WithCancel(ctx)
	return &LogExporter{
		provider: provider,
		logger:   logger,
		ctx:      runCtx,
		cancel:   cancel,
	}, nil
}

// Shutdown 优雅关闭并 flush 在途日志。
func (l *LogExporter) Shutdown(ctx context.Context) error {
	l.cancel()
	l.wg.Wait()
	return l.provider.Shutdown(ctx)
}

// ZapCore 返回接入日志链路的 zapcore.Core（异步、有界缓冲、非阻塞）。
func (l *LogExporter) ZapCore(level zapcore.LevelEnabler) zapcore.Core {
	ch := make(chan logEntry, 4096)
	l.wg.Add(1)
	go l.worker(ch)
	return &otelLogCore{
		level:  level,
		fields: nil,
		ch:     ch,
	}
}

type logEntry struct {
	time   time.Time
	level  zapcore.Level
	msg    string
	fields []zapcore.Field
	caller string
}

func (l *LogExporter) worker(ch <-chan logEntry) {
	defer l.wg.Done()
	for {
		select {
		case <-l.ctx.Done():
			// 优雅停机：drain 剩余条目后退出
			for {
				select {
				case entry, ok := <-ch:
					if !ok {
						return
					}
					l.emit(entry)
				default:
					return
				}
			}
		case entry, ok := <-ch:
			if !ok {
				return
			}
			l.emit(entry)
		}
	}
}

func (l *LogExporter) emit(entry logEntry) {
	record := otelog.Record{}
	record.SetTimestamp(entry.time)
	record.SetSeverityText(entry.level.String())
	record.SetSeverity(zapLevelToSeverity(entry.level))
	record.SetBody(attribute.StringValue(entry.msg))

	n := len(entry.fields)
	if entry.caller != "" {
		n++
	}
	attrs := make([]attribute.KeyValue, 0, n)
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range entry.fields {
		f.AddTo(enc)
	}
	for k, v := range enc.Fields {
		attrs = append(attrs, attribute.String(k, fmt.Sprint(v)))
	}
	if entry.caller != "" {
		attrs = append(attrs, attribute.String("caller", entry.caller))
	}
	if len(attrs) > 0 {
		record.AddAttributes(attrs...)
	}
	l.logger.Emit(l.ctx, record)
}

// otelLogCore zapcore.Core 桥接实现：写入有界 channel。
type otelLogCore struct {
	level  zapcore.LevelEnabler
	fields []zapcore.Field
	ch     chan<- logEntry
}

func (c *otelLogCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

func (c *otelLogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(append([]zapcore.Field(nil), c.fields...), fields...)
	return &clone
}

func (c *otelLogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *otelLogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	entry := logEntry{
		time:   ent.Time,
		level:  ent.Level,
		msg:    ent.Message,
		fields: append(append([]zapcore.Field(nil), c.fields...), fields...),
	}
	if ent.Caller.Defined {
		entry.caller = ent.Caller.TrimmedPath()
	}
	select {
	case c.ch <- entry:
	default:
		// 缓冲满时丢弃，避免日志风暴阻塞调用方
	}
	return nil
}

func (c *otelLogCore) Sync() error { return nil }

func zapLevelToSeverity(level zapcore.Level) otelog.Severity {
	switch {
	case level <= zapcore.DebugLevel:
		return otelog.SeverityDebug
	case level < zapcore.WarnLevel:
		return otelog.SeverityInfo
	case level < zapcore.ErrorLevel:
		return otelog.SeverityWarn
	default:
		return otelog.SeverityError
	}
}

var _ zapcore.Core = (*otelLogCore)(nil)
