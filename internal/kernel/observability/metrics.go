package observability

import (
	"database/sql"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DBStats DB 连接池指标
	dbStatsMaxOpenConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "max_open_connections",
		Help:      "Maximum number of open database connections",
	}, []string{"name"})

	dbStatsOpenConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "open_connections",
		Help:      "Number of established database connections",
	}, []string{"name"})

	dbStatsInUseConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "in_use_connections",
		Help:      "Number of in-use database connections",
	}, []string{"name"})

	dbStatsIdleConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "idle_connections",
		Help:      "Number of idle database connections",
	}, []string{"name"})

	dbStatsWaitCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "wait_count_total",
		Help:      "Total number of connections waited for",
	}, []string{"name"})

	dbStatsWaitDuration = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "db",
		Name:      "wait_duration_seconds_total",
		Help:      "Total time blocked waiting for a new connection",
	}, []string{"name"})

	// RuntimeStats 运行时指标
	runtimeGoroutines = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "runtime",
		Name:      "goroutines",
		Help:      "Number of goroutines",
	})

	runtimeMemoryAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "runtime",
		Name:      "memory_alloc_bytes",
		Help:      "Allocated memory bytes",
	})

	runtimeMemorySys = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "runtime",
		Name:      "memory_sys_bytes",
		Help:      "Memory obtained from OS",
	})

	runtimeGCCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "runtime",
		Name:      "gc_count_total",
		Help:      "Total number of GC cycles",
	})
)

// DBCollector DB 连接池指标收集器
type DBCollector struct {
	db   *sql.DB
	name string
}

// NewDBCollector 创建 DB 指标收集器
func NewDBCollector(db *sql.DB, name string) *DBCollector {
	return &DBCollector{db: db, name: name}
}

// Collect 收集 DB 连接池指标（定期调用）
func (c *DBCollector) Collect() {
	if c.db == nil {
		return
	}
	stats := c.db.Stats()
	dbStatsMaxOpenConnections.WithLabelValues(c.name).Set(float64(stats.MaxOpenConnections))
	dbStatsOpenConnections.WithLabelValues(c.name).Set(float64(stats.OpenConnections))
	dbStatsInUseConnections.WithLabelValues(c.name).Set(float64(stats.InUse))
	dbStatsIdleConnections.WithLabelValues(c.name).Set(float64(stats.Idle))
	dbStatsWaitCount.WithLabelValues(c.name).Add(float64(stats.WaitCount))
	dbStatsWaitDuration.WithLabelValues(c.name).Add(stats.WaitDuration.Seconds())
}

// CollectRuntime 收集运行时指标（定期调用）
func CollectRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	runtimeGoroutines.Set(float64(runtime.NumGoroutine()))
	runtimeMemoryAlloc.Set(float64(memStats.Alloc))
	runtimeMemorySys.Set(float64(memStats.Sys))
	runtimeGCCount.Add(float64(memStats.NumGC))
}
