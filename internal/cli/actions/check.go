package actions

import (
	"context"

	"jimu/tools/generator"
	"jimu/tools/generator/manifest"
)

// CheckInput is the shared input for direct and TUI lifecycle checks.
type CheckInput struct {
	Dir, SourceRoot string
	Framework       *manifest.Framework
	Deep            bool
}

// RunCheck executes the shared read-only lifecycle check.
func RunCheck(ctx context.Context, in CheckInput) (*generator.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir := in.Dir
	if dir == "" {
		dir = "."
	}
	return generator.CheckProject(generator.CheckOptions{
		Dir:       dir,
		From:      in.SourceRoot,
		Framework: in.Framework,
		Deep:      in.Deep,
	})
}
