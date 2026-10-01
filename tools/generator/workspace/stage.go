// Package workspace owns the filesystem transaction around manifest execution.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/render"
	"jimu/tools/generator/report"
	"jimu/tools/internal/projectmetrics"
)

const manifestRel = ".jimu/manifest.json"

type CreateOptions struct {
	Target     string
	SourceRoot string
	Module     string
	Force      bool
	NoTidy     bool
	DryRun     bool
	Report     bool

	// NoSelfCheck is reserved for package-level tests and callers that provide
	// their own verification. The CLI keeps the default fail-closed behavior.
	NoSelfCheck bool
}

type UpdateOptions struct {
	ProjectRoot string
	SourceRoot  string
	Module      string
	Force       bool
	DryRun      bool
	NoTidy      bool
	NoSelfCheck bool
	Report      bool
}

type Result struct {
	Target    string
	Module    string
	Shape     string
	Files     []string
	Changed   []string
	Removed   []string
	FileCount int
}

func Create(doc manifest.Document, opts CreateOptions) (Result, error) {
	executionPlan, err := plan.Build(doc)
	if err != nil {
		return Result{}, err
	}
	if opts.SourceRoot == "" || opts.Target == "" {
		return Result{}, fmt.Errorf("source root and target are required")
	}
	module := moduleFromDocument(doc, opts.Module)
	if module == "" {
		return Result{}, fmt.Errorf("module is required")
	}
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return Result{}, fmt.Errorf("resolve target: %w", err)
	}
	files := append(slices.Clone(executionPlan.GeneratedFiles), manifestRel)
	sort.Strings(files)
	result := Result{Target: target, Module: module, Shape: doc.Selection.Shape, Files: files, FileCount: len(files)}
	if opts.DryRun {
		return result, nil
	}
	if err := validateCreateTarget(target, opts.Force); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, fmt.Errorf("create target parent: %w", err)
	}
	staging, err := createStage(target)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	rendered, err := render.Execute(render.Context{
		SourceRoot:  opts.SourceRoot,
		Destination: staging,
		Module:      module,
		Plan:        executionPlan,
	})
	if err != nil {
		return Result{}, err
	}
	doc.GeneratedFiles = slices.Clone(rendered)
	doc.Digest = ""
	doc.Digest, err = manifest.Digest(doc)
	if err != nil {
		return Result{}, err
	}
	if !opts.NoTidy {
		if err := Tidy(staging); err != nil {
			return Result{}, fmt.Errorf("go mod tidy: %w", err)
		}
	}
	if !opts.NoSelfCheck {
		if err := SelfCheck(staging); err != nil {
			return Result{}, fmt.Errorf("self check: %w", err)
		}
	}
	if err := manifest.Write(filepath.Join(staging, manifestRel), doc); err != nil {
		return Result{}, err
	}
	if opts.Report {
		if err := writeReport(staging, module, doc); err != nil {
			return Result{}, err
		}
	}
	if err := replaceDirectory(target, staging); err != nil {
		return Result{}, err
	}
	sort.Strings(rendered)
	result.Files = append(rendered, manifestRel)
	sort.Strings(result.Files)
	result.FileCount = len(result.Files)
	return result, nil
}

func Update(previous manifest.Document, next manifest.Document, opts UpdateOptions) (Result, error) {
	if err := manifest.Validate(previous); err != nil {
		return Result{}, fmt.Errorf("validate previous manifest: %w", err)
	}
	executionPlan, err := plan.Build(next)
	if err != nil {
		return Result{}, err
	}
	if opts.ProjectRoot == "" || opts.SourceRoot == "" {
		return Result{}, fmt.Errorf("project root and source root are required")
	}
	projectRoot, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return Result{}, fmt.Errorf("resolve project root: %w", err)
	}
	current, err := LoadProjectManifest(projectRoot)
	if err != nil {
		return Result{}, err
	}
	if !opts.Force && previous.Digest != "" && current.Digest != previous.Digest {
		return Result{}, fmt.Errorf("project manifest changed since it was loaded; pass --force to continue")
	}
	module := moduleFromDocument(next, opts.Module)
	if module == "" {
		return Result{}, fmt.Errorf("module is required")
	}
	staging, err := createStage(projectRoot)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if _, err := render.Execute(render.Context{
		SourceRoot:  opts.SourceRoot,
		Destination: staging,
		Module:      module,
		Plan:        executionPlan,
	}); err != nil {
		return Result{}, err
	}
	rendered := listFiles(staging)
	next.GeneratedFiles = slices.Clone(rendered)
	next.Digest = ""
	next.Digest, err = manifest.Digest(next)
	if err != nil {
		return Result{}, err
	}
	if err := mergeConfigs(projectRoot, staging, next.Merges); err != nil {
		return Result{}, err
	}
	if err := manifest.Write(filepath.Join(staging, manifestRel), next); err != nil {
		return Result{}, err
	}
	if opts.Report {
		if err := writeReport(staging, module, next); err != nil {
			return Result{}, err
		}
	}
	managedFiles := slices.Clone(next.GeneratedFiles)
	managedFiles = append(managedFiles, manifestRel)
	if opts.Report {
		managedFiles = append(managedFiles, report.ReportPath)
	}
	changed, removed, err := diffGenerated(projectRoot, staging, previous.GeneratedFiles, managedFiles)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Target:    projectRoot,
		Module:    module,
		Shape:     next.Selection.Shape,
		Changed:   changed,
		Removed:   removed,
		FileCount: len(changed),
	}
	result.Files = append(slices.Clone(next.GeneratedFiles), manifestRel)
	sort.Strings(result.Files)
	if opts.DryRun || (len(changed) == 0 && len(removed) == 0) {
		return result, nil
	}
	if err := installStaged(projectRoot, staging, changed, removed); err != nil {
		return Result{}, err
	}
	return result, nil
}

func listFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func moduleFromDocument(doc manifest.Document, override string) string {
	if override != "" {
		return override
	}
	for _, action := range doc.Rewrites {
		if action.Kind == "module" && action.To != "" {
			return action.To
		}
	}
	return ""
}

func validateCreateTarget(target string, force bool) error {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("target %s is not a directory", target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("target directory %s is not empty; pass --force to replace it", target)
	}
	if len(entries) > 0 && force {
		if _, err := LoadProjectManifest(target); err != nil {
			return fmt.Errorf("target directory %s is not a generated project: --force requires %s: %w", target, manifestRel, err)
		}
	}
	return nil
}

func createStage(target string) (string, error) {
	stage, err := os.MkdirTemp(filepath.Dir(target), filepath.Base(target)+".tmp-")
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}
	return stage, nil
}

func writeReport(root, module string, doc manifest.Document) error {
	spec := projectmetrics.ReportSpec{
		Name:         doc.Selection.Shape,
		Capabilities: doc.Report.Capabilities,
		Routes:       doc.Report.Routes,
		Migrations:   len(doc.Report.Migrations),
		Tables:       len(doc.Report.Tables),
	}
	metrics, err := report.Measure(root, module, spec, nil)
	if err != nil {
		return fmt.Errorf("measure generated report: %w", err)
	}
	metadata := report.Metadata{
		Module:        module,
		Shape:         doc.Selection.Shape,
		Capabilities:  slices.Clone(doc.Selection.Capabilities),
		MigrationOnly: capabilityNames(doc, true, false),
		DomainOnly:    capabilityNames(doc, false, true),
		Drivers:       driverNames(doc.Selection.Drivers),
		Assets:        assetNames(doc),
	}
	_, err = report.Write(root, metrics, metadata, assetRoots(doc))
	return err
}

func capabilityNames(doc manifest.Document, migrationOnly, domainOnly bool) []string {
	values := make([]string, 0)
	for _, capability := range doc.Capabilities {
		if capability.MigrationOnly == migrationOnly && capability.DomainOnly == domainOnly {
			if migrationOnly || domainOnly {
				values = append(values, capability.Name)
			}
		}
	}
	sort.Strings(values)
	return values
}

func driverNames(drivers map[string][]string) []string {
	names := make([]string, 0, len(drivers))
	for name, values := range drivers {
		if len(values) == 0 {
			continue
		}
		names = append(names, name+"="+strings.Join(values, ","))
	}
	sort.Strings(names)
	return names
}

func assetNames(doc manifest.Document) []string {
	names := make([]string, 0, len(doc.Assets))
	for _, asset := range doc.Assets {
		names = append(names, asset.Destination)
	}
	sort.Strings(names)
	return names
}

func assetRoots(doc manifest.Document) []string {
	seen := map[string]bool{}
	var roots []string
	for _, asset := range doc.Assets {
		root := strings.SplitN(asset.Destination, "/", 2)[0]
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	return roots
}
