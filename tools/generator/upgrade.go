package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/preflight"
	"jimu/tools/generator/render"
	upgradeplan "jimu/tools/generator/upgrade"
	"jimu/tools/generator/workspace"
)

// UpgradeOptions controls an explicit upgrade. Dir is the generated project;
// From optionally overrides framework source discovery.
type UpgradeOptions struct {
	Dir       string
	From      string
	NoTidy    bool
	Framework *manifest.Framework
	// IncludeDiff adds local-to-target text diffs to the read-only report.
	IncludeDiff bool
}

// UpgradeResult is the planner output. Applying it is the responsibility of
// ApplyUpgrade; the plan itself never mutates the project.
type UpgradeResult struct {
	Dir        string
	Module     string
	Resolution frameworkmanifest.Resolution
	Previous   manifest.Document
	Target     manifest.Document
	Report     upgradeplan.Report
}

// UpgradeApplyResult is the committed transaction result. It embeds the plan
// that was applied so callers can print conflicts or a summary.
type UpgradeApplyResult struct {
	*UpgradeResult
	Changed         []string
	ManifestUpdated bool
	Verified        bool
}

// ErrUpgradeConflicts marks a plan that cannot be applied without manual
// resolution.
var ErrUpgradeConflicts = errors.New("upgrade plan has conflicts")

// upgradeVerify is replaceable in package tests; the CLI always uses the
// workspace build/test gate.
var upgradeVerify = workspace.Verify

type plannedUpgrade struct {
	UpgradeResult
	staging string
}

// PlanUpgrade renders the target template into a temporary directory and
// classifies every managed path against the manifest baseline and current
// project content. It never writes to the project directory.
func PlanUpgrade(opts UpgradeOptions) (*UpgradeResult, error) {
	planned, err := planUpgrade(opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(planned.staging) }()
	return &planned.UpgradeResult, nil
}

// ApplyUpgrade plans and commits an explicit upgrade. It refuses conflicts
// before writing, backs up every touched file, installs changed files, verifies
// the upgraded project, and updates the manifest as one transaction.
func ApplyUpgrade(opts UpgradeOptions) (*UpgradeApplyResult, error) {
	planned, err := planUpgrade(opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(planned.staging) }()
	result := &UpgradeApplyResult{UpgradeResult: &planned.UpgradeResult}
	if planned.Report.HasConflicts() {
		return result, fmt.Errorf("%w: %d", ErrUpgradeConflicts, planned.Report.Count(upgradeplan.Conflict))
	}
	if err := planned.Report.VerifyLocal(planned.Dir); err != nil {
		return result, err
	}
	applied, err := workspace.ApplyUpgrade(workspace.UpgradeApplyOptions{
		ProjectRoot: planned.Dir,
		StagingRoot: planned.staging,
		Install:     upgradeInstallPaths(planned.Report),
		Target:      planned.Target,
		Verify:      upgradeVerify,
	})
	if err != nil {
		return result, err
	}
	result.Changed = applied.Changed
	result.ManifestUpdated = applied.ManifestUpdated
	result.Verified = applied.Verified
	return result, nil
}

func planUpgrade(opts UpgradeOptions) (*plannedUpgrade, error) {
	dir := absPath(opts.Dir)
	previous, err := workspace.LoadProjectManifest(dir)
	if err != nil {
		return nil, err
	}
	root, err := resolveSourceRoot(opts.From)
	if err != nil {
		return nil, err
	}
	request, err := upgradeRequest(previous, root, dir, opts.Framework)
	if err != nil {
		return nil, err
	}
	target, resolution, err := frameworkmanifest.ExportDetailed(request)
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(target); err != nil {
		return nil, fmt.Errorf("validate target manifest: %w", err)
	}
	if err := preflight.ValidateWithOverlay(target, frameworkmanifest.CompositionDescriptors()); err != nil {
		return nil, fmt.Errorf("target composition invalid: %w", err)
	}
	executionPlan, err := plan.Build(target)
	if err != nil {
		return nil, fmt.Errorf("build target plan: %w", err)
	}
	staging, err := os.MkdirTemp("", "jimu-upgrade-plan-")
	if err != nil {
		return nil, fmt.Errorf("create upgrade staging directory: %w", err)
	}
	keepStaging := false
	defer func() {
		if !keepStaging {
			_ = os.RemoveAll(staging)
		}
	}()
	rendered, err := render.Execute(render.Context{
		SourceRoot:  root,
		Destination: staging,
		Module:      request.Module,
		Plan:        executionPlan,
	})
	if err != nil {
		return nil, fmt.Errorf("render target template: %w", err)
	}
	if !opts.NoTidy {
		if _, statErr := os.Stat(filepath.Join(staging, "go.mod")); statErr == nil {
			if err := workspace.Tidy(staging); err != nil {
				return nil, fmt.Errorf("target go mod tidy: %w", err)
			}
		} else if !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("inspect staged go.mod: %w", statErr)
		}
	}
	if err := workspace.MergeConfigs(dir, staging, target.Merges); err != nil {
		return nil, fmt.Errorf("merge target config: %w", err)
	}
	previousForClassify := previous
	targetFiles := slices.Clone(rendered)
	_, reportErr := os.Stat(filepath.Join(dir, filepath.FromSlash(reportRelPath)))
	switch {
	case reportErr == nil:
		if err := workspace.WriteReport(staging, request.Module, target); err != nil {
			return nil, fmt.Errorf("refresh generated report: %w", err)
		}
		reportHashes, err := manifest.HashFiles(dir, []string{reportRelPath})
		if err != nil {
			return nil, fmt.Errorf("hash generated report: %w", err)
		}
		previousForClassify.GeneratedFiles = append(slices.Clone(previous.GeneratedFiles), reportRelPath)
		previousForClassify.GeneratedFileSHA256 = make(map[string]string, len(previous.GeneratedFileSHA256)+1)
		for path, hash := range previous.GeneratedFileSHA256 {
			previousForClassify.GeneratedFileSHA256[path] = hash
		}
		previousForClassify.GeneratedFileSHA256[reportRelPath] = reportHashes[reportRelPath]
		targetFiles = append(targetFiles, reportRelPath)
	case !os.IsNotExist(reportErr):
		return nil, fmt.Errorf("inspect generated report: %w", reportErr)
	}
	planReport, err := upgradeplan.Classify(upgradeplan.Options{
		ProjectRoot: dir,
		Previous:    previousForClassify,
		TargetRoot:  staging,
		TargetFiles: targetFiles,
		Merged:      upgradeMergeDestinations(target.Merges),
		IncludeDiff: opts.IncludeDiff,
	})
	if err != nil {
		return nil, err
	}
	target.GeneratedFiles = slices.Clone(rendered)
	target.GeneratedFileSHA256, err = manifest.HashFiles(staging, rendered)
	if err != nil {
		return nil, err
	}
	target.Digest = ""
	digest, err := manifest.Digest(target)
	if err != nil {
		return nil, err
	}
	target.Digest = digest
	keepStaging = true
	return &plannedUpgrade{UpgradeResult: UpgradeResult{
		Dir:        dir,
		Module:     request.Module,
		Resolution: resolution,
		Previous:   previous,
		Target:     target,
		Report:     planReport,
	}, staging: staging}, nil
}

func upgradeMergeDestinations(actions []manifest.MergeAction) []string {
	var out []string
	for _, action := range actions {
		if action.Strategy == "sections" {
			out = append(out, action.Destination)
		}
	}
	sort.Strings(out)
	return slices.Compact(out)
}

func upgradeInstallPaths(report upgradeplan.Report) []string {
	var out []string
	for _, entry := range report.Entries {
		if entry.Action == upgradeplan.ActionInstall {
			out = append(out, entry.Path)
		}
	}
	sort.Strings(out)
	return out
}

func upgradeRequest(previous manifest.Document, root, projectDir string, framework *manifest.Framework) (frameworkmanifest.Request, error) {
	module := moduleFromManifest(previous)
	if module == "" {
		module, _ = moduleOf(filepath.Join(projectDir, "go.mod"))
	}
	if module == "" {
		return frameworkmanifest.Request{}, fmt.Errorf("cannot determine project module: manifest has no module rewrite and go.mod is unreadable")
	}
	request := frameworkmanifest.Request{Root: root, Module: module, Framework: framework}
	switch {
	case previous.Selection.Template != "":
		request.Template = previous.Selection.Template
	case previous.Selection.Profile != "":
		request.Profile = previous.Selection.Profile
	case previous.Composition != nil && len(previous.Composition.Requested) > 0:
		request.With = strings.Join(previous.Composition.Requested, ",")
		request.Shape = previous.Selection.Shape
		request.Drivers = cloneSelectedDrivers(previous.Composition.Drivers)
	default:
		return frameworkmanifest.Request{}, fmt.Errorf("manifest has no reproducible selection (template, profile or composition.requested)")
	}
	return request, nil
}

func cloneSelectedDrivers(drivers map[string][]string) map[string][]string {
	if len(drivers) == 0 {
		return nil
	}
	out := make(map[string][]string, len(drivers))
	for name, values := range drivers {
		out[name] = slices.Clone(values)
	}
	return out
}
