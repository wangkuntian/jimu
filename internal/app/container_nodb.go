//go:build jimu_nodb

package app

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/health"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/metrics"
	"jimu/internal/platform/scheduler"
)

// DBHandle is intentionally opaque in no-DB builds.
type DBHandle = any

type dbState struct{}

func newDBState(*config.Config, *logger.Logger, map[contract.RuntimeService]bool) (dbState, error) {
	return dbState{}, nil
}

func (dbState) handle() DBHandle { return nil }

func (dbState) close() error { return nil }

func (dbState) mysqlStore() (scheduler.Store, error) {
	return nil, fmt.Errorf("scheduler store mysql requires a mysql runtime service")
}

func (dbState) collector() (*metrics.DBCollector, error) { return nil, nil }

func (dbState) readiness() ([]health.Checker, error) { return nil, nil }
