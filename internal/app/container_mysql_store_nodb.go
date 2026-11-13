//go:build !jimu_lean && jimu_nodb

package app

import (
	"fmt"

	"jimu/internal/platform/scheduler"
)

func (dbState) mysqlStore() (scheduler.Store, error) {
	return nil, fmt.Errorf("scheduler store mysql requires a mysql runtime service")
}
