package workspace

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type PruneOptions struct {
	Module   string
	Tests    string
	Runtime  map[string]bool
	TestDeps map[string]bool
}

type PruneResult struct {
	Removed     []string
	RemovedDirs []string
}

// PrunePackages removes Go packages that are not reachable from the runtime
// closure. Test-only packages survive only when their non-test files stay
// inside the runtime closure.
func PrunePackages(dir string, opts PruneOptions) (PruneResult, error) {
	if opts.Module == "" {
		return PruneResult{}, fmt.Errorf("prune module is empty")
	}
	if !slices.Contains([]string{"selected", "all", "none"}, opts.Tests) {
		return PruneResult{}, fmt.Errorf("prune tests mode %q is unknown", opts.Tests)
	}
	dirs, err := collectGoDirs(dir)
	if err != nil {
		return PruneResult{}, err
	}
	var result PruneResult
	for _, packageDir := range dirs {
		importPath := importPathForDir(dir, packageDir, opts.Module)
		keep := opts.Runtime[importPath]
		if !keep && opts.Tests == "all" && opts.TestDeps[importPath] {
			keep, err = nonTestImportsSafe(packageDir, opts.Module, opts.Runtime)
			if err != nil {
				return PruneResult{}, err
			}
		}
		files, err := os.ReadDir(packageDir)
		if err != nil {
			return PruneResult{}, fmt.Errorf("read package dir %s: %w", packageDir, err)
		}
		for _, entry := range files {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			rel, err := filepath.Rel(dir, filepath.Join(packageDir, entry.Name()))
			if err != nil {
				return PruneResult{}, err
			}
			rel = filepath.ToSlash(rel)
			remove := !keep
			if strings.HasSuffix(entry.Name(), "_test.go") {
				switch opts.Tests {
				case "none":
					remove = true
				case "selected":
					remove = !opts.Runtime[importPath]
				case "all":
					remove = !keep
				}
			}
			if !remove {
				continue
			}
			if err := os.Remove(filepath.Join(packageDir, entry.Name())); err != nil {
				return PruneResult{}, fmt.Errorf("remove %s: %w", rel, err)
			}
			result.Removed = append(result.Removed, rel)
		}
	}
	result.RemovedDirs, err = removeEmptyDirs(dir)
	if err != nil {
		return PruneResult{}, err
	}
	slices.Sort(result.Removed)
	return result, nil
}

func collectGoDirs(root string) ([]string, error) {
	seen := map[string]bool{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			seen[filepath.Dir(path)] = true
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk generated tree: %w", err)
	}
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs, nil
}

func importPathForDir(root, dir, module string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return module
	}
	return module + "/" + filepath.ToSlash(rel)
}

func nonTestImportsSafe(dir, module string, runtime map[string]bool) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read package dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			return false, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if belongsToModule(path, module) && !runtime[path] {
				return false, nil
			}
		}
	}
	return true, nil
}

func removeEmptyDirs(root string) ([]string, error) {
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk directories: %w", err)
	}
	slices.SortFunc(dirs, func(a, b string) int {
		return strings.Compare(b, a)
	})
	var removed []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read empty dir %s: %w", dir, err)
		}
		if len(entries) != 0 {
			continue
		}
		if err := os.Remove(dir); err != nil {
			return nil, fmt.Errorf("remove empty dir %s: %w", dir, err)
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil, err
		}
		removed = append(removed, filepath.ToSlash(rel))
	}
	slices.Sort(removed)
	return removed, nil
}
