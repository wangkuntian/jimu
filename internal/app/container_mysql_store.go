//go:build !jimu_lean && !jimu_nodb

package app

import (
	"fmt"

	"jimu/internal/platform/scheduler"
)

func (s dbState) mysqlStore() (scheduler.Store, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("scheduler store mysql requires a mysql runtime service")
	}
	return scheduler.NewMySQLStore(s.conn), nil
}
