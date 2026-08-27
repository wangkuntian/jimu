// internal/platform/reporter/reporter.go
package reporter

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
)

// LogFunc 结构化日志输出函数（msg + 交替 key/value）。
// 由调用方注入（如 logger.Logger.Errorw），reporter 包不依赖日志实现，避免 import 循环。
type LogFunc func(msg string, keysAndValues ...interface{})

// ReporterConfig 错误追踪上报配置
type ReporterConfig struct {
	// 是否启用错误上报（false 时 Reporter 为空实现，零开销）
	Enabled bool `mapstructure:"enabled"`
	// Sentry DSN。配置后错误上报到 Sentry；留空则仅本地日志。
	DSN string `mapstructure:"dsn"`
	// 环境标签（如 production/staging），Sentry 事件环境
	Environment string `mapstructure:"environment"`
	// SampleRate 采样率 0-1，1.0 表示全部上报
	SampleRate float64 `mapstructure:"sample_rate"`
}

// DefaultReporterConfig 返回默认错误上报配置
func DefaultReporterConfig() ReporterConfig {
	return ReporterConfig{
		SampleRate: 1.0,
	}
}

// Reporter 错误追踪上报接口：捕获业务错误与未恢复 panic 到聚合平台。
type Reporter interface {
	// Report 上报单个错误。attrs 为附加上下文（key/value 交替）。
	// 实现必须容忍任意失败（不阻塞调用方），推荐异步发送。
	Report(ctx context.Context, err error, attrs ...string)
	// Flush 阻塞等待在途上报完成（优雅停机时调用）。
	Flush(timeout time.Duration) bool
}

// emptyReporter 空实现：未启用时零开销。
type emptyReporter struct{}

func (emptyReporter) Report(context.Context, error, ...string) {}
func (emptyReporter) Flush(time.Duration) bool                 { return true }

// logReporter 本地日志实现：结构化错误日志（含 trace_id），总可用。
type logReporter struct {
	logf LogFunc
}

func (r *logReporter) Report(ctx context.Context, err error, attrs ...string) {
	if r == nil || r.logf == nil || err == nil {
		return
	}
	fields := make([]interface{}, 0, len(attrs)+2)
	fields = append(fields, "error", err.Error())
	if len(attrs) > 0 {
		fields = append(fields, toPairs(attrs)...)
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		fields = append(fields, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	r.logf("error reported", fields...)
}

func (r *logReporter) Flush(time.Duration) bool { return true }

// sentryReporter Sentry 实现：DSN 配置后启用，HTTP/队列/定时任务错误统一上报。
type sentryReporter struct {
	env string
}

// NewSentryReporter 初始化 Sentry 客户端。DSN 为空或初始化失败返回错误（调用方决定回退日志）。
func NewSentryReporter(cfg ReporterConfig) (*sentryReporter, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("sentry reporter: dsn is empty")
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		SampleRate:       cfg.SampleRate,
		AttachStacktrace: true,
	}); err != nil {
		return nil, fmt.Errorf("sentry init: %w", err)
	}
	return &sentryReporter{env: cfg.Environment}, nil
}

func (r *sentryReporter) Report(ctx context.Context, err error, attrs ...string) {
	if err == nil {
		return
	}
	event := sentry.NewEvent()
	event.Level = sentry.LevelError
	event.Message = err.Error()
	event.Environment = r.env
	// 额外上下文展开为标签（限制长度避免 event 过大）
	if len(attrs) > 0 {
		if event.Tags == nil {
			event.Tags = make(map[string]string)
		}
		pairs := toPairs(attrs)
		for i := 0; i+1 < len(pairs); i += 2 {
			k := fmt.Sprint(pairs[i])
			v := fmt.Sprint(pairs[i+1])
			if len(v) > 500 {
				v = v[:500]
			}
			event.Tags[k] = v
		}
	}
	hub := sentry.CurrentHub().Clone()
	hub.CaptureEvent(event)
}

func (r *sentryReporter) Flush(timeout time.Duration) bool {
	if r == nil {
		return true
	}
	return sentry.Flush(timeout)
}

// toPairs 将交替 key/value 展开为 slice（容错奇数长度）。
func toPairs(attrs []string) []interface{} {
	out := make([]interface{}, 0, len(attrs))
	for i := 0; i+1 < len(attrs); i += 2 {
		out = append(out, attrs[i], attrs[i+1])
	}
	return out
}

// NewReporter 按配置创建错误上报器：
//   - 未启用：空实现
//   - DSN 配置成功：Sentry + 本地日志双通道
//   - DSN 配置失败：仅本地日志，并记录初始化错误（不阻断启动）
//
// logf 为结构化日志回调（如 logger.Errorw）；为 nil 时日志通道静默。
func NewReporter(cfg ReporterConfig, logf LogFunc) Reporter {
	if !cfg.Enabled {
		return emptyReporter{}
	}
	logRep := &logReporter{logf: logf}
	if cfg.DSN == "" {
		return logRep
	}
	sr, err := NewSentryReporter(cfg)
	if err != nil {
		if logf != nil {
			logf("sentry disabled, falling back to log reporter", "error", err.Error())
		}
		return logRep
	}
	return &dualReporter{log: logRep, sentry: sr}
}

// dualReporter 同时写日志与 Sentry。
type dualReporter struct {
	log    Reporter
	sentry Reporter
}

func (d *dualReporter) Report(ctx context.Context, err error, attrs ...string) {
	d.log.Report(ctx, err, attrs...)
	d.sentry.Report(ctx, err, attrs...)
}

func (d *dualReporter) Flush(timeout time.Duration) bool {
	a := d.log.Flush(timeout)
	b := d.sentry.Flush(timeout)
	return a && b
}

var _ Reporter = emptyReporter{}
var _ Reporter = (*logReporter)(nil)
var _ Reporter = (*sentryReporter)(nil)
var _ Reporter = (*dualReporter)(nil)
