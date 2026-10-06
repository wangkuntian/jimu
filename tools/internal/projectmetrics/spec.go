package projectmetrics

import (
	"fmt"
	"slices"
)

// ReportSpec contains framework facts that the neutral metrics engine cannot
// infer from a generated source tree without importing framework packages.
type ReportSpec struct {
	Name         string
	Capabilities []string
	Routes       int
	Migrations   int
	Tables       int
}

// Measure combines static report facts with source-tree measurements. The
// engine deliberately knows nothing about assembly, profiles, or descriptors.
func Measure(root, modulePath string, spec ReportSpec, overlay map[string][]byte) (Metrics, error) {
	files, lines, heavy, err := ClosureSize(root, modulePath, overlay)
	if err != nil {
		return Metrics{}, fmt.Errorf("measure closure of %s: %w", spec.Name, err)
	}
	return Metrics{
		Profile:      spec.Name,
		Routes:       spec.Routes,
		Migrations:   spec.Migrations,
		Tables:       spec.Tables,
		Files:        files,
		Lines:        lines,
		HeavyDeps:    heavy,
		Capabilities: slices.Clone(spec.Capabilities),
	}, nil
}
