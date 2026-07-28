package observability

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterDebugRoutes 注册 pprof 和 metrics 端点
// 路由: /debug/pprof/* 和 /debug/metrics
func RegisterDebugRoutes(r *gin.RouterGroup) {
	// Go 内置 pprof
	r.GET("/pprof/*any", gin.WrapH(http.DefaultServeMux))

	// 自定义 metrics
	r.GET("/metrics", func(c *gin.Context) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		goroutines := runtime.NumGoroutine()

		c.JSON(200, gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"memory": gin.H{
				"alloc":       memStats.Alloc,
				"total_alloc": memStats.TotalAlloc,
				"sys":         memStats.Sys,
				"gc_count":    memStats.NumGC,
				"heap_alloc":  memStats.HeapAlloc,
				"heap_sys":    memStats.HeapSys,
				"heap_idle":   memStats.HeapIdle,
				"heap_inuse":  memStats.HeapInuse,
			},
			"goroutines": goroutines,
			"cpu": gin.H{
				"num_cpu":       runtime.NumCPU(),
				"num_goroutine": goroutines,
			},
			"gc": gin.H{
				"pause_total_ns": memStats.PauseTotalNs,
				"num_gc":         memStats.NumGC,
				"gc_cpu_fraction": memStats.GCCPUFraction,
			},
		})
	})
}
