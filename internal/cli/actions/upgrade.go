package actions

import (
	"context"
	"errors"

	"jimu/tools/generator"
	"jimu/tools/generator/manifest"
)

var (
	ErrUpgradeModes           = errors.New("--dry-run and --check are mutually exclusive")
	ErrUpgradeDiagnosticModes = errors.New("--json and --diff require --dry-run or --check")
)

// UpgradeInput is the shared input for direct and TUI upgrades.
type UpgradeInput struct {
	Dir, SourceRoot string
	Framework       *manifest.Framework
	NoTidy          bool
	DryRun, Check   bool
	IncludeDiff     bool
}

// RunUpgrade applies an upgrade or returns a read-only plan. Exactly one of the
// two results is non-nil on success.
func RunUpgrade(ctx context.Context, in UpgradeInput) (*generator.UpgradeResult, *generator.UpgradeApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if in.DryRun && in.Check {
		return nil, nil, ErrUpgradeModes
	}
	if in.IncludeDiff && !in.DryRun && !in.Check {
		return nil, nil, ErrUpgradeDiagnosticModes
	}
	dir := in.Dir
	if dir == "" {
		dir = "."
	}
	if !in.DryRun && !in.Check {
		applied, err := generator.ApplyUpgrade(generator.UpgradeOptions{
			Dir: dir, From: in.SourceRoot, Framework: in.Framework, NoTidy: in.NoTidy,
		})
		return nil, applied, err
	}
	result, err := generator.PlanUpgrade(generator.UpgradeOptions{
		Dir: dir, From: in.SourceRoot, Framework: in.Framework, NoTidy: in.NoTidy,
		IncludeDiff: in.IncludeDiff,
	})
	return result, nil, err
}
