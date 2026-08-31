package observability

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func newTestPusher(t *testing.T, registry *prometheus.Registry) *MetricsPusher {
	t.Helper()
	pusher := &MetricsPusher{
		registry:  registry,
		interval:  time.Second,
		startTime: time.Now(),
	}
	return pusher
}

func TestGatherToOTLP_CounterAndGauge(t *testing.T) {
	registry := prometheus.NewRegistry()
	counter := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "requests_total",
		Help:      "total requests",
	}, []string{"method"})
	gauge := promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "temperature",
		Help:      "current temperature",
	})
	counter.WithLabelValues("GET").Add(5)
	gauge.Set(36.5)

	pusher := newTestPusher(t, registry)
	rm, err := pusher.gatherToOTLP()
	if err != nil {
		t.Fatalf("gatherToOTLP() error = %v", err)
	}
	if rm == nil {
		t.Fatal("gatherToOTLP() returned nil")
	}
	if len(rm.ScopeMetrics) != 1 || len(rm.ScopeMetrics[0].Metrics) != 2 {
		t.Fatalf("unexpected metrics count: %+v", rm.ScopeMetrics)
	}

	var counterSeen, gaugeSeen bool
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "test_requests_total":
			sum, ok := m.Data.(metricdata.Sum[float64])
			if !ok || !sum.IsMonotonic {
				t.Fatalf("counter not monotonic sum: %T", m.Data)
			}
			if len(sum.DataPoints) != 1 || sum.DataPoints[0].Value != 5 {
				t.Fatalf("counter value = %+v", sum.DataPoints)
			}
			counterSeen = true
		case "test_temperature":
			g, ok := m.Data.(metricdata.Gauge[float64])
			if !ok || len(g.DataPoints) != 1 || g.DataPoints[0].Value != 36.5 {
				t.Fatalf("gauge value = %+v", g.DataPoints)
			}
			gaugeSeen = true
		}
	}
	if !counterSeen || !gaugeSeen {
		t.Fatalf("counterSeen=%v gaugeSeen=%v", counterSeen, gaugeSeen)
	}
}

func TestGatherToOTLP_Histogram(t *testing.T) {
	registry := prometheus.NewRegistry()
	hist := promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
		Namespace: "test",
		Name:      "latency_seconds",
		Help:      "request latency",
		Buckets:   []float64{0.1, 0.5, 1},
	})
	hist.Observe(0.2)
	hist.Observe(0.8)

	pusher := newTestPusher(t, registry)
	rm, err := pusher.gatherToOTLP()
	if err != nil {
		t.Fatalf("gatherToOTLP() error = %v", err)
	}
	if len(rm.ScopeMetrics[0].Metrics) != 1 {
		t.Fatalf("unexpected metrics: %+v", rm.ScopeMetrics[0].Metrics)
	}
	h, ok := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected histogram, got %T", rm.ScopeMetrics[0].Metrics[0].Data)
	}
	if len(h.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(h.DataPoints))
	}
	dp := h.DataPoints[0]
	if dp.Count != 2 {
		t.Fatalf("count = %d, want 2", dp.Count)
	}
	if len(dp.Bounds) != 3 || len(dp.BucketCounts) != 3 {
		t.Fatalf("bounds = %v, counts = %v", dp.Bounds, dp.BucketCounts)
	}
	if dp.BucketCounts[1] != 1 || dp.BucketCounts[2] != 2 {
		t.Fatalf("cumulative buckets = %v, want [0 1 2]", dp.BucketCounts)
	}
}

func TestGatherToOTLP_EmptyRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	pusher := newTestPusher(t, registry)
	rm, err := pusher.gatherToOTLP()
	if err != nil {
		t.Fatalf("gatherToOTLP() error = %v", err)
	}
	if rm != nil {
		t.Fatalf("expected nil for empty registry, got %+v", rm)
	}
}
