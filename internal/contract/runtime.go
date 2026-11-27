package contract

import (
	"context"
	"net/http"
	"time"
)

// HTTPClient 是能力使用统一出站 HTTP 客户端的最小接口。
type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// MetricsCollector 是内核周期采集指标的最小接口。
type MetricsCollector interface {
	Collect()
}

// Reporter 是错误上报接口；实现必须容忍任意失败。
type Reporter interface {
	Report(ctx context.Context, err error, attrs ...string)
	Flush(timeout time.Duration) bool
}

// Scheduler 是组合根注册、恢复和停止定时任务所需的内核接口。
type Scheduler interface {
	JobRegistry
	Component
	ErrorSource
	AddNamedFunc(id, name, spec string, cmd func()) error
	RestoreFromStore(ctx context.Context, cmdFactory func(id string) func()) ([]string, error)
	Jobs() []JobInfo
	TriggerJob(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
}

// JobInfo 是调用方观察到的已注册任务信息。
type JobInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	LastRun    time.Time `json:"last_run,omitempty"`
	LastStatus string    `json:"last_status"`
	LastError  string    `json:"last_error,omitempty"`
	RunCount   int64     `json:"run_count"`
}
