package generator

import (
	"fmt"
	"slices"
	"strings"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/report"
	"jimu/tools/generator/workspace"
	"jimu/tools/internal/projectmetrics"
)

const reportRelPath = report.ReportPath

// Report measures a generated project from its manifest. Framework-specific
// facts are already captured in manifest.report, so rerunning the report does
// not load the framework assembly or catalog.
func Report(root string) (*projectmetrics.Metrics, error) {
	doc, err := workspace.LoadProjectManifest(root)
	if err != nil {
		return nil, err
	}
	module := manifestModule(doc)
	if module == "" {
		return nil, fmt.Errorf("manifest does not declare a module rewrite")
	}
	spec := projectmetrics.ReportSpec{
		Name:         doc.Report.Name,
		Capabilities: doc.Report.Capabilities,
		Routes:       doc.Report.Routes,
		Migrations:   len(doc.Report.Migrations),
		Tables:       len(doc.Report.Tables),
	}
	metrics, err := report.Measure(root, module, spec, nil)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// WriteReport keeps the root facade stable for callers that already have a
// CapabilitySet. The persisted manifest remains the source of truth.
func WriteReport(dir string, metrics projectmetrics.Metrics, _ CapabilitySet) error {
	doc, err := workspace.LoadProjectManifest(dir)
	if err != nil {
		return err
	}
	module := manifestModule(doc)
	if module == "" {
		return fmt.Errorf("manifest does not declare a module rewrite")
	}
	metadata := report.Metadata{
		Module:       module,
		Shape:        doc.Selection.Shape,
		Capabilities: doc.Selection.Capabilities,
		Drivers:      flattenDrivers(doc.Selection.Drivers),
		Assets:       manifestAssetNames(doc),
	}
	for _, capability := range doc.Capabilities {
		if capability.MigrationOnly {
			metadata.MigrationOnly = append(metadata.MigrationOnly, capability.Name)
		}
		if capability.DomainOnly {
			metadata.DomainOnly = append(metadata.DomainOnly, capability.Name)
		}
	}
	slices.Sort(metadata.MigrationOnly)
	slices.Sort(metadata.DomainOnly)
	_, err = report.Write(dir, metrics, metadata, manifestAssetRoots(doc))
	return err
}

func manifestAssetRoots(doc manifest.Document) []string {
	seen := map[string]bool{}
	var roots []string
	for _, asset := range doc.Assets {
		root := strings.SplitN(asset.Destination, "/", 2)[0]
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	slices.Sort(roots)
	return roots
}

func manifestModule(doc manifest.Document) string {
	for _, rewrite := range doc.Rewrites {
		if rewrite.Kind == "module" {
			return rewrite.To
		}
	}
	return ""
}
