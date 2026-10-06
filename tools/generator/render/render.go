// Package render executes a manifest-derived generation plan.
// It does not import Jimu framework packages; all framework facts arrive through plan.Plan.
package render

import (
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/tools/generator/plan"
)

type Context struct {
	SourceRoot  string
	Destination string
	Module      string
	Plan        plan.Plan
}

func Execute(ctx Context) ([]string, error) {
	if err := plan.Validate(ctx.Plan); err != nil {
		return nil, err
	}
	if ctx.SourceRoot == "" || ctx.Destination == "" {
		return nil, fmt.Errorf("source root and destination are required")
	}
	if err := os.MkdirAll(ctx.Destination, 0o755); err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}
	for _, action := range ctx.Plan.Copy {
		if err := copyTree(ctx.SourceRoot, ctx.Destination, action.Source, action.Destination, action.Include, action.Exclude); err != nil {
			if action.Optional && os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	for _, action := range ctx.Plan.Templates {
		if err := renderTemplate(ctx.SourceRoot, ctx.Destination, action); err != nil {
			return nil, err
		}
	}
	for _, action := range ctx.Plan.Merges {
		if err := mergeConfig(ctx.SourceRoot, ctx.Destination, action); err != nil {
			return nil, err
		}
	}
	if err := narrowDrivers(ctx.Destination, ctx.Plan.Selection.Drivers); err != nil {
		return nil, err
	}
	for _, action := range ctx.Plan.Rewrites {
		if err := applyRewrite(ctx.Destination, ctx.Module, action); err != nil {
			return nil, err
		}
	}
	// Deployment assets contain framework-owned names such as /opt/jimu. Copy them
	// after module rewriting so those names remain literal asset content.
	for _, action := range ctx.Plan.Assets {
		if err := copyTree(ctx.SourceRoot, ctx.Destination, action.Source, action.Destination, action.Include, action.Exclude); err != nil {
			return nil, err
		}
	}
	if err := pruneTests(ctx.Destination, ctx.Module, ctx.Plan.Prune); err != nil {
		return nil, err
	}
	if err := formatTree(ctx.Destination); err != nil {
		return nil, err
	}
	return generatedFiles(ctx.Destination), nil
}

func generatedFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(files)
	return files
}

func formatTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), ".tmpl") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(content)
		if err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}
		return os.WriteFile(path, formatted, 0o644)
	})
}
