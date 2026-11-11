package observability

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otlploggrpc "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelog "go.opentelemetry.io/otel/log"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
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
		otlploggrpc.WithHeaders(otlpHeaders(cfg, cfg.LogsStreamName)),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp logs exporter: %w", err)
	}

	// 资源属性带实例标识：OpenObserve 支持按实例（host/pid）过滤日志
	host, _ := os.Hostname()
	provider := logsdk.NewLoggerProvider(
		logsdk.WithResource(sdkresource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(defaulted(cfg.ServiceName, "jimu")),
			semconv.ServiceVersion(defaulted(cfg.ServiceVersion, "dev")),
			semconv.ServiceInstanceID(host+":"+strconv.Itoa(os.Getpid())),
			semconv.HostName(host),
		)),
		logsdk.WithProcessor(logsdk.NewBatchProcessor(exporter)),
	)
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

	// 关联追踪：WithContext 注入的 trace_id/span_id 字符串字段解析为 span context，
	// sdk/log 的 Emit 会从 ctx 自动填充 LogRecord 的 traceID/spanID（OTLP 专用字段），
	// OpenObserve 据此把日志挂到对应 trace。字段本身仍保留为普通属性供检索。
	ctx := l.ctx
	if tid, sid, ok := traceIDsFromFields(entry.fields); ok {
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    tid,
			SpanID:     sid,
			TraceFlags: trace.FlagsSampled,
		})
		if sc.IsValid() {
			ctx = trace.ContextWithSpanContext(ctx, sc)
		}
	}

	attrs := make([]attribute.KeyValue, 0, len(entry.fields)+1)
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range entry.fields {
		f.AddTo(enc)
	}
	for k, v := range enc.Fields {
		attrs = append(attrs, attributeFromValue(k, v))
	}
	if entry.caller != "" {
		attrs = append(attrs, attribute.String("caller", entry.caller))
	}
	if len(attrs) > 0 {
		record.AddAttributes(attrs...)
	}
	l.logger.Emit(ctx, record)
}

// traceIDsFromFields 从 logger.WithContext 注入的 trace_id/span_id 字符串字段
// 解析追踪标识。解析失败（非 hex）时静默跳过，字段仍作为普通属性保留。
func traceIDsFromFields(fields []zapcore.Field) (trace.TraceID, trace.SpanID, bool) {
	var tid trace.TraceID
	var sid trace.SpanID
	haveTrace, haveSpan := false, false
	for _, f := range fields {
		if f.Type != zapcore.StringType {
			continue
		}
		switch f.Key {
		case "trace_id":
			if id, err := trace.TraceIDFromHex(f.String); err == nil {
				tid, haveTrace = id, true
			}
		case "span_id":
			if id, err := trace.SpanIDFromHex(f.String); err == nil {
				sid, haveSpan = id, true
			}
		}
	}
	return tid, sid, haveTrace && haveSpan
}

// attributeFromValue 保留字段类型，避免全部退化为字符串：
// 数值/布尔保持数值类型（OpenObserve 可范围查询/聚合），
// 时长固定为纳秒数值（与 zap.Duration 字段语义一致），
// 其余类型（map/slice/struct）兜底为字符串。
func attributeFromValue(k string, v interface{}) attribute.KeyValue {
	switch tv := v.(type) {
	case string:
		return attribute.String(k, tv)
	case int:
		return attribute.Int(k, tv)
	case int64:
		return attribute.Int64(k, tv)
	case float64:
		return attribute.Float64(k, tv)
	case bool:
		return attribute.Bool(k, tv)
	case time.Duration:
		return attribute.Int64(k, int64(tv))
	case time.Time:
		return attribute.String(k, tv.Format(time.RFC3339Nano))
	case []string:
		return attribute.StringSlice(k, tv)
	case []bool:
		return attribute.BoolSlice(k, tv)
	case []int:
		return attribute.IntSlice(k, tv)
	case []int64:
		return attribute.Int64Slice(k, tv)
	case []float64:
		return attribute.Float64Slice(k, tv)
	default:
		return attribute.String(k, fmt.Sprint(v))
	}
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
