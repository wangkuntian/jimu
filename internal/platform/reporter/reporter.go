// internal/platform/reporter/reporter.go
package reporter

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// LogFunc 结构化日志输出函数（msg + 交替 key/value）。
// 由调用方注入（如 logger.Logger.Errorw），reporter 包不依赖日志实现，避免 import 循环。
type LogFunc func(msg string, keysAndValues ...interface{})

// ReporterConfig 错误上报配置
type ReporterConfig struct {
	// 是否启用错误上报（false 时 Reporter 为空实现，零开销）
	Enabled bool `mapstructure:"enabled"`
}

// DefaultReporterConfig 返回默认错误上报配置
func DefaultReporterConfig() ReporterConfig {
	return ReporterConfig{Enabled: true}
}

// Reporter 错误上报接口：捕获业务错误与未恢复 panic 到聚合平台。
// 实现通过结构化错误日志输出（含 trace_id），日志链路接入 OpenObserve 后
// 错误自然进入可观测平台，配合 OpenObserve 告警覆盖错误监控场景。
type Reporter interface {
	// Report 上报单个错误。attrs 为附加上下文（key/value 交替）。
	// 实现必须容忍任意失败（不阻塞调用方）。
	Report(ctx context.Context, err error, attrs ...string)
	// Flush 阻塞等待在途上报完成（优雅停机时调用）。
	Flush(timeout time.Duration) bool
}

// emptyReporter 空实现：未启用时零开销。
type emptyReporter struct{}

func (emptyReporter) Report(context.Context, error, ...string) {}
func (emptyReporter) Flush(time.Duration) bool                 { return true }

// logReporter 本地结构化日志实现（含 trace_id），总可用。
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

// toPairs 将交替 key/value 展开为 slice（容错奇数长度）。
func toPairs(attrs []string) []interface{} {
	out := make([]interface{}, 0, len(attrs))
	for i := 0; i+1 < len(attrs); i += 2 {
		out = append(out, attrs[i], attrs[i+1])
	}
	return out
}

// NewReporter 按配置创建错误上报器：
//   - 未启用：空实现（零开销）
//   - 启用：本地结构化日志（日志链路接入 OpenObserve 后错误自动汇聚）
func NewReporter(cfg ReporterConfig, logf LogFunc) Reporter {
	if !cfg.Enabled {
		return emptyReporter{}
	}
	return &logReporter{logf: logf}
}

var _ Reporter = emptyReporter{}
var _ Reporter = (*logReporter)(nil)
