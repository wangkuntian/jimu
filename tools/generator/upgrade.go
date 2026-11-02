package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/preflight"
	"jimu/tools/generator/render"
	upgradeplan "jimu/tools/generator/upgrade"
	"jimu/tools/generator/workspace"
)

// UpgradeOptions controls a read-only upgrade plan. Dir is the generated
// project; From optionally overrides framework source discovery.
type UpgradeOptions struct {
	Dir    string
	From   string
	NoTidy bool
}

// UpgradeResult is the planner output. It is safe to inspect but must not be
// used to mutate the project until the P0-5 transaction exists.
type UpgradeResult struct {
	Dir        string
	Module     string
	Resolution frameworkmanifest.Resolution
	Previous   manifest.Document
	Target     manifest.Document
	Report     upgradeplan.Report
}

// PlanUpgrade renders the target template into a temporary directory and
// classifies every managed path against the manifest baseline and current
// project content. It never writes to the project directory.
func PlanUpgrade(opts UpgradeOptions) (*UpgradeResult, error) {
	dir := absPath(opts.Dir)
	previous, err := workspace.LoadProjectManifest(dir)
	if err != nil {
		return nil, err
	}
	root := opts.From
	if root == "" {
		root, err = frameworkRoot()
		if err != nil {
			return nil, err
		}
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve framework source root: %w", err)
	}
	request, err := upgradeRequest(previous, root, dir)
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
	defer func() { _ = os.RemoveAll(staging) }()
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
	report, err := upgradeplan.Classify(upgradeplan.Options{
		ProjectRoot: dir,
		Previous:    previous,
		TargetRoot:  staging,
		TargetFiles: rendered,
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
	return &UpgradeResult{
		Dir:        dir,
		Module:     request.Module,
		Resolution: resolution,
		Previous:   previous,
		Target:     target,
		Report:     report,
	}, nil
}

func upgradeRequest(previous manifest.Document, root, projectDir string) (frameworkmanifest.Request, error) {
	module := moduleFromManifest(previous)
	if module == "" {
		module, _ = moduleOf(filepath.Join(projectDir, "go.mod"))
	}
	if module == "" {
		return frameworkmanifest.Request{}, fmt.Errorf("cannot determine project module: manifest has no module rewrite and go.mod is unreadable")
	}
	request := frameworkmanifest.Request{Root: root, Module: module}
	switch {
	case previous.Selection.Template != "":
		request.Template = previous.Selection.Template
	case previous.Selection.Profile != "":
		request.Profile = previous.Selection.Profile
	case previous.Composition != nil && len(previous.Composition.Requested) > 0:
		request.With = strings.Join(previous.Composition.Requested, ",")
		request.Shape = previous.Selection.Shape
	default:
		return frameworkmanifest.Request{}, fmt.Errorf("manifest has no reproducible selection (template, profile or composition.requested)")
	}
	return request, nil
}
