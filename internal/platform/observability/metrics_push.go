package observability

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel/attribute"
	otlpmetricgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// MetricsPusher 将 Prometheus registry 指标定期转换为 OTLP 并推送到
// OpenObserve（OTLP/gRPC 端点）。现有 promauto 指标定义保持不变，
// /metrics 端点继续可用，推送作为补充通道。
type MetricsPusher struct {
	exporter *otlpmetricgrpc.Exporter
	registry *prometheus.Registry
	resource *resource.Resource
	interval time.Duration
	// startTime 固定为 pusher 创建时刻，作为所有 cumulative 序列的起始时间
	startTime time.Time

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMetricsPusher 创建指标推送器。interval <= 0 时默认 15s。
func NewMetricsPusher(ctx context.Context, cfg TracingConfig, registry *prometheus.Registry) (*MetricsPusher, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp metrics exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(defaulted(cfg.ServiceName, "jimu")),
		semconv.ServiceVersion(defaulted(cfg.ServiceVersion, "dev")),
	)

	interval := time.Duration(cfg.MetricsInterval) * time.Second
	if interval <= 0 {
		interval = 15 * time.Second
	}

	runCtx, cancel := context.WithCancel(ctx)
	return &MetricsPusher{
		exporter:  exporter,
		registry:  registry,
		resource:  res,
		interval:  interval,
		startTime: time.Now(),
		ctx:       runCtx,
		cancel:    cancel,
	}, nil
}

// Start 启动后台推送循环（每 interval 推送一次）。
func (p *MetricsPusher) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				p.push(context.Background())
				return
			case <-ticker.C:
				p.push(p.ctx)
			}
		}
	}()
}

// Shutdown 停止推送并 flush 最后一批指标。
func (p *MetricsPusher) Shutdown(ctx context.Context) error {
	p.cancel()
	p.wg.Wait()
	return p.exporter.Shutdown(ctx)
}

// push 采集一次 Prometheus registry 并推送 OTLP 数据。
// 失败仅记录（metricdata 无日志通道，静默丢弃，下轮重试）。
func (p *MetricsPusher) push(ctx context.Context) {
	rm, err := p.gatherToOTLP()
	if err != nil || rm == nil {
		return
	}
	_ = p.exporter.Export(ctx, rm)
}

func (p *MetricsPusher) gatherToOTLP() (*metricdata.ResourceMetrics, error) {
	families, err := p.registry.Gather()
	if err != nil {
		return nil, fmt.Errorf("gather prometheus metrics: %w", err)
	}

	now := time.Now()
	metrics := make([]metricdata.Metrics, 0, len(families))
	for _, mf := range families {
		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			points := make([]metricdata.DataPoint[float64], 0, len(mf.GetMetric()))
			for _, m := range mf.GetMetric() {
				points = append(points, metricdata.DataPoint[float64]{
					Attributes: attrsFromLabels(m.GetLabel()),
					StartTime:  p.startTime,
					Time:       now,
					Value:      m.GetCounter().GetValue(),
				})
			}
			metrics = append(metrics, metricdata.Metrics{
				Name:        mf.GetName(),
				Description: mf.GetHelp(),
				Data: metricdata.Sum[float64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  points,
				},
			})
		case dto.MetricType_GAUGE:
			points := make([]metricdata.DataPoint[float64], 0, len(mf.GetMetric()))
			for _, m := range mf.GetMetric() {
				points = append(points, metricdata.DataPoint[float64]{
					Attributes: attrsFromLabels(m.GetLabel()),
					Time:       now,
					Value:      m.GetGauge().GetValue(),
				})
			}
			metrics = append(metrics, metricdata.Metrics{
				Name:        mf.GetName(),
				Description: mf.GetHelp(),
				Data: metricdata.Gauge[float64]{
					DataPoints: points,
				},
			})
		case dto.MetricType_HISTOGRAM:
			points := make([]metricdata.HistogramDataPoint[float64], 0, len(mf.GetMetric()))
			for _, m := range mf.GetMetric() {
				h := m.GetHistogram()
				bounds := make([]float64, 0, len(h.GetBucket()))
				counts := make([]uint64, 0, len(h.GetBucket()))
				for _, b := range h.GetBucket() {
					bounds = append(bounds, b.GetUpperBound())
					counts = append(counts, b.GetCumulativeCount())
				}
				count := uint64(0)
				if h.GetSampleCount() > 0 {
					count = h.GetSampleCount()
				} else if len(counts) > 0 {
					count = counts[len(counts)-1]
				}
				sum := h.GetSampleSum()
				if math.IsNaN(sum) {
					sum = 0
				}
				points = append(points, metricdata.HistogramDataPoint[float64]{
					Attributes:   attrsFromLabels(m.GetLabel()),
					Time:         now,
					Count:        count,
					Bounds:       bounds,
					BucketCounts: counts,
					Sum:          sum,
				})
			}
			metrics = append(metrics, metricdata.Metrics{
				Name:        mf.GetName(),
				Description: mf.GetHelp(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints:  points,
				},
			})
		default:
			// Summary/UNTYPED 暂不转换（当前应用未使用）
		}
	}

	if len(metrics) == 0 {
		return nil, nil
	}
	return &metricdata.ResourceMetrics{
		Resource: p.resource,
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Scope:   instrumentation.Scope{Name: "jimu"},
				Metrics: metrics,
			},
		},
	}, nil
}

func attrsFromLabels(labels []*dto.LabelPair) attribute.Set {
	if len(labels) == 0 {
		return attribute.Set{}
	}
	kvs := make([]attribute.KeyValue, 0, len(labels))
	for _, l := range labels {
		kvs = append(kvs, attribute.String(l.GetName(), l.GetValue()))
	}
	return attribute.NewSet(kvs...)
}

func defaulted(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
