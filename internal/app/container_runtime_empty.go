//go:build jimu_lean

package app

import (
	"context"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/logger"
)

// runtimeDeps 在 lean 构建中为空：不编入调度器、出站 HTTP、错误上报或指标。
type runtimeDeps struct {
	jobRegistry contract.JobRegistry
	scheduler   contract.Scheduler
	httpClient  contract.HTTPClient
	reporter    contract.Reporter
	collector   contract.MetricsCollector
	stopFn      func(context.Context) error
	collectFn   func()
}

func newRuntimeDeps(*config.Config, *logger.Logger, *CapabilityConfigs, dbState, redisState) (runtimeDeps, error) {
	return runtimeDeps{
		stopFn:    func(context.Context) error { return nil },
		collectFn: func() {},
	}, nil
}
