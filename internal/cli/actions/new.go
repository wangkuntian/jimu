package actions

import (
	"context"
	"errors"

	"jimu/tools/generator"
	"jimu/tools/generator/manifest"
)

var (
	ErrProfileWithMutuallyExclusive = errors.New("--profile and --with are mutually exclusive")
	ErrShapeRequiresWith            = errors.New("--shape is only valid with --with")
)

// NewInput is the shared input for direct and TUI scaffolding.
type NewInput struct {
	Dir, Module, Profile, With, Shape, DBDriver string
	Tests, Tools, Deploy                        string
	NoTidy, DryRun, Force, Report               bool
	SourceRoot                                  string
	Framework                                   *manifest.Framework
}

// RunNew validates selectors and delegates to the generator.
func RunNew(ctx context.Context, in NewInput) (*generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Profile != "" && in.With != "" {
		return nil, ErrProfileWithMutuallyExclusive
	}
	if in.Shape != "" && in.Profile != "" {
		return nil, ErrShapeRequiresWith
	}
	return generator.NewProject(generator.NewOptions{
		Dir:        in.Dir,
		Template:   in.Profile,
		With:       in.With,
		Shape:      in.Shape,
		Module:     in.Module,
		DBDriver:   in.DBDriver,
		Tests:      in.Tests,
		Tools:      in.Tools,
		Deploy:     in.Deploy,
		NoTidy:     in.NoTidy,
		DryRun:     in.DryRun,
		Force:      in.Force,
		Report:     in.Report,
		SourceRoot: in.SourceRoot,
		Framework:  in.Framework,
	})
}
