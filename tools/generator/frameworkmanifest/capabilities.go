package frameworkmanifest

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"jimu/internal/contract"
)

func capabilityClosure(root string, declared []string, requires map[string][]string, known map[string]bool) (closure, roots []string, err error) {
	if len(declared) == 0 {
		return nil, nil, nil
	}
	dir := filepath.Join(root, "internal", "capabilities")
	patterns := make([]string, 0, len(declared))
	for _, name := range declared {
		patterns = append(patterns, "./"+name)
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:   dir,
		Tests: false,
	}, patterns...)
	if err != nil {
		return nil, nil, fmt.Errorf("load capability import graph under %s: %w", filepath.ToSlash(dir), err)
	}
	prefix := "jimu/internal/capabilities/"
	found := map[string]bool{}
	rootSet := map[string]bool{}
	var loadErrors []string
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, loadErr := range pkg.Errors {
			loadErrors = append(loadErrors, loadErr.Error())
		}
		rest, ok := strings.CutPrefix(pkg.PkgPath, prefix)
		if !ok {
			return
		}
		name, sub, hasSub := strings.Cut(rest, "/")
		if hasSub && (sub == "domain" || strings.HasPrefix(sub, "domain/")) {
			return
		}
		if !known[name] {
			return
		}
		found[name] = true
		if !hasSub {
			rootSet[name] = true
		}
	})
	if len(loadErrors) > 0 {
		return nil, nil, fmt.Errorf("load capability import graph: %s", strings.Join(loadErrors, "; "))
	}
	for changed := true; changed; {
		changed = false
		for name := range found {
			for _, dependency := range requires[name] {
				if !known[dependency] {
					return nil, nil, fmt.Errorf("capability %q requires unknown capability %q", name, dependency)
				}
				if !found[dependency] {
					found[dependency] = true
					changed = true
				}
			}
		}
	}
	closure = make([]string, 0, len(found))
	for name := range found {
		closure = append(closure, name)
	}
	roots = make([]string, 0, len(rootSet))
	for name := range rootSet {
		roots = append(roots, name)
	}
	slices.Sort(closure)
	slices.Sort(roots)
	return closure, roots, nil
}

func migrationFiles(name string, migrations fs.FS) []string {
	if migrations == nil {
		return []string{}
	}
	var out []string
	_ = fs.WalkDir(migrations, "migrations/mysql", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		out = append(out, filepath.ToSlash(filepath.Join("internal/capabilities", name, path)))
		return nil
	})
	slices.Sort(out)
	return out
}

func descriptorByName(descriptors []contract.Descriptor) map[string]contract.Descriptor {
	out := make(map[string]contract.Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		out[descriptor.Name] = descriptor
	}
	return out
}
