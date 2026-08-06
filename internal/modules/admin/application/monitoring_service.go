package application

import (
	"context"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdminMonitoringService 运维监控服务
type AdminMonitoringService struct {
	startTime time.Time
	version   string
	env       string
	redis     *redis.Client
}

// NewAdminMonitoringService 创建运维监控服务
func NewAdminMonitoringService(version, env string, rdb *redis.Client) *AdminMonitoringService {
	return &AdminMonitoringService{
		startTime: time.Now(),
		version:   version,
		env:       env,
		redis:     rdb,
	}
}

// SystemStatus 系统状态
type SystemStatus struct {
	Version      string      `json:"version"`
	Environment  string      `json:"environment"`
	StartTime    time.Time   `json:"start_time"`
	Uptime       string      `json:"uptime"`
	NumGoroutine int         `json:"num_goroutine"`
	NumCPU       int         `json:"num_cpu"`
	Memory       MemoryStats `json:"memory"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
	HeapAlloc  uint64 `json:"heap_alloc"`
	HeapInuse  uint64 `json:"heap_inuse"`
}

// GetStatus 获取系统状态
func (s *AdminMonitoringService) GetStatus() SystemStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemStatus{
		Version:      s.version,
		Environment:  s.env,
		StartTime:    s.startTime,
		Uptime:       time.Since(s.startTime).String(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		Memory: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
			HeapAlloc:  memStats.HeapAlloc,
			HeapInuse:  memStats.HeapInuse,
		},
	}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Redis    bool `json:"redis"`
	Database bool `json:"database"`
}

// GetHealth 获取依赖健康状态
func (s *AdminMonitoringService) GetHealth(ctx context.Context) HealthStatus {
	health := HealthStatus{}
	if s.redis != nil {
		_, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		health.Redis = s.redis.Ping(ctx).Err() == nil
	}
	return health
}
