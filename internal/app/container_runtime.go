//go:build !jimu_lean

package app

import (
	"context"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/httpclient"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/metrics"
	"jimu/internal/platform/reporter"
	"jimu/internal/platform/scheduler"
)

// runtimeDeps 是内核运行期组件集合。标准构建包含调度器、出站 HTTP、
// 错误上报与指标；lean 构建由 container_runtime_empty.go 提供空实现。
type runtimeDeps struct {
	jobRegistry contract.JobRegistry
	scheduler   contract.Scheduler
	httpClient  contract.HTTPClient
	reporter    contract.Reporter
	collector   contract.MetricsCollector
	stopFn      func(context.Context) error
	collectFn   func()
}

func newRuntimeDeps(
	cfg *config.Config,
	log *logger.Logger,
	capCfgs *CapabilityConfigs,
	database dbState,
	redis redisState,
) (runtimeDeps, error) {
	var schedulerCfg scheduler.Config
	if c, ok := SectionOf[*scheduler.Config](capCfgs, scheduler.ConfigKey); ok {
		schedulerCfg = *c
	}
	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if schedulerCfg.Store == scheduler.StoreMySQL {
		store, err := database.mysqlStore()
		if err != nil {
			return runtimeDeps{}, err
		}
		schedStore = store
	}
	var sched *scheduler.CronScheduler
	if schedulerCfg.Store == scheduler.StoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, redis.locker())
	} else {
		sched = scheduler.NewWithStore(log, schedStore, nil)
	}

	httpClient := httpclient.New(httpclient.Config{
		TimeoutSec:      cfg.HTTPClient.TimeoutSec,
		MaxRetries:      cfg.HTTPClient.MaxRetries,
		RetryIntervalMS: cfg.HTTPClient.RetryIntervalMS,
		RateLimitRate:   cfg.HTTPClient.RateLimitRate,
		RateLimitBurst:  cfg.HTTPClient.RateLimitBurst,
	})
	collector, err := database.collector()
	if err != nil {
		return runtimeDeps{}, err
	}
	errorReporter := reporter.NewReporter(cfg.ErrorReport.Enabled, log.Errorw)
	return runtimeDeps{
		jobRegistry: sched,
		scheduler:   sched,
		httpClient:  httpClient,
		reporter:    errorReporter,
		collector:   collector,
		stopFn: func(context.Context) error {
			if errorReporter != nil {
				errorReporter.Flush(5 * time.Second)
			}
			return nil
		},
		collectFn: func() {
			if collector != nil {
				collector.Collect()
			}
			metrics.CollectRuntime()
		},
	}, nil
}
