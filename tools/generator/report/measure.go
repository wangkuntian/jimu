// Package report renders a single generated project's compile-surface report.
// Framework-specific facts arrive through manifest-derived ReportSpec values.
package report

import (
	"jimu/tools/internal/projectmetrics"
)

type Metadata struct {
	Module         string
	Shape          string
	Capabilities   []string
	MigrationOnly  []string
	DomainOnly     []string
	Drivers        []string
	Assets         []string
	GeneratedFiles int
	AssetFiles     int
}

func Measure(root, modulePath string, spec projectmetrics.ReportSpec, overlay map[string][]byte) (projectmetrics.Metrics, error) {
	return projectmetrics.Measure(root, modulePath, spec, overlay)
}
