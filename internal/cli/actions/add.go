package actions

import (
	"context"

	"jimu/tools/generator"
	"jimu/tools/generator/manifest"
)

// AddInput is the shared input for direct and TUI capability additions.
type AddInput struct {
	Name, Driver, Dir, Module string
	SourceRoot                string
	Framework                 *manifest.Framework
	Force, DryRun             bool
}

// RunAddCapability adds a capability to an existing generated project.
func RunAddCapability(ctx context.Context, in AddInput) (*generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return generator.AddCapability(generator.AddOptions{
		Name:      in.Name,
		Driver:    in.Driver,
		From:      in.SourceRoot,
		Dir:       in.Dir,
		Module:    in.Module,
		Force:     in.Force,
		DryRun:    in.DryRun,
		Framework: in.Framework,
	})
}
