//go:build !jimu_no_otel

package app

import (
	"context"
	"errors"
	"log"

	"jimu/internal/config"
	"jimu/internal/kernel/observability"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

// otelState 持有 OTel 追踪/指标/日志出口与 DB 指标采集器。
// 只在带 OTel 的构建里引用 OTel SDK 与导出器。
type otelState struct {
	tracerProvider *sdktrace.TracerProvider
	metricsPusher  *observability.MetricsPusher
	logExporter    *observability.LogExporter
	extra          []zapcore.Core
}

func newOTELState(cfg *config.Config) otelState {
	state := otelState{}
	if cfg.OTEL.Enabled && cfg.OTEL.LogsEnabled {
		exporter, err := observability.NewLogExporter(context.Background(), cfg.OTEL)
		if err != nil {
			log.Printf("openobserve logs exporter init failed: %v", err)
		} else {
			state.logExporter = exporter
			state.extra = append(state.extra, exporter.ZapCore(zapcore.DebugLevel))
		}
	}
	return state
}

func (s *otelState) extraCores() []zapcore.Core { return s.extra }

func (s *otelState) shutdown(ctx context.Context) error {
	var result error
	if s.tracerProvider != nil {
		result = errors.Join(result, observability.ShutdownTracing(ctx, s.tracerProvider))
	}
	if s.metricsPusher != nil {
		result = errors.Join(result, s.metricsPusher.Shutdown(ctx))
	}
	if s.logExporter != nil {
		result = errors.Join(result, s.logExporter.Shutdown(ctx))
	}
	return result
}
