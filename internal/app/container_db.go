//go:build !jimu_nodb

package app

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/db"
	"jimu/internal/platform/health"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/metrics"
	"jimu/internal/platform/scheduler"

	"gorm.io/gorm"
)

// DBHandle is the concrete database handle in DB-enabled builds.
type DBHandle = *gorm.DB

type dbState struct {
	conn *gorm.DB
}

func newDBState(cfg *config.Config, log *logger.Logger, services map[contract.RuntimeService]bool) (dbState, error) {
	if err := db.InitSnowflake(cfg.ID.WorkerID); err != nil {
		return dbState{}, err
	}
	if !services[contract.RuntimeServiceMySQL] {
		return dbState{}, nil
	}
	conn, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return dbState{}, err
	}
	return dbState{conn: conn}, nil
}

func (s dbState) handle() DBHandle { return s.conn }

func (s dbState) close() error {
	if s.conn == nil {
		return nil
	}
	sqlDB, err := s.conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s dbState) mysqlStore() (scheduler.Store, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("scheduler store mysql requires a mysql runtime service")
	}
	return scheduler.NewMySQLStore(s.conn), nil
}

func (s dbState) collector() (*metrics.DBCollector, error) {
	if s.conn == nil {
		return nil, nil
	}
	sqlDB, err := s.conn.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	return metrics.NewDBCollector(sqlDB, "primary"), nil
}

func (s dbState) readiness() ([]health.Checker, error) {
	if s.conn == nil {
		return nil, nil
	}
	sqlDB, err := s.conn.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	return []health.Checker{health.NewSQLChecker(sqlDB)}, nil
}
