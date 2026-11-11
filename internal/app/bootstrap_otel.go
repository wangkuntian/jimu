//go:build !jimu_no_otel

package app

import (
	"context"

	"jimu/internal/platform/observability"

	"github.com/prometheus/client_golang/prometheus"
)

// 独立成文件：生成器按能力集裁剪时，不需要可观测性的项目可以整份排除，
// 从而不把 OTel 与 Prometheus 推送到二进制里。bootstrap.go 里有一个
// 同名的空实现，本文件的 init() 会把它替换成真实实现。
func init() { initObservability = setupObservability }

func setupObservability(container *Container) error {
	cfg := container.Config

	tp, err := observability.InitTracing(context.Background(), cfg.OTEL)
	if err != nil {
		return err
	}
	container.otel.tracerProvider = tp

	// 指标推送（OTLP/gRPC → OpenObserve）：基于 Prometheus 默认 registry。
	if cfg.OTEL.Enabled && cfg.OTEL.MetricsEnabled {
		reg, ok := prometheus.DefaultRegisterer.(*prometheus.Registry)
		if !ok {
			container.Logger.Errorw("openobserve metrics pusher init failed", "error", "default registerer type mismatch")
		} else {
			pusher, err := observability.NewMetricsPusher(context.Background(), cfg.OTEL, reg)
			if err != nil {
				container.Logger.Errorw("openobserve metrics pusher init failed", "error", err.Error())
			} else {
				pusher.Start()
				container.otel.metricsPusher = pusher
			}
		}
	}

	// 启用 OTel 时插桩 DB/Redis，捕获查询子 span。
	// 必须在 InitTracing 之后：插件创建时固化全局 TracerProvider。
	if cfg.OTEL.Enabled {
		if err := container.db.instrumentOTel(); err != nil {
			return err
		}
		if err := instrumentRedisTracing(container); err != nil {
			return err
		}
	}
	return nil
}
