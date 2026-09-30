package render

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"jimu/tools/generator/manifest"
)

func pruneTests(root, module string, rules []manifest.PruneRule) error {
	for _, rule := range rules {
		if rule.Kind != "test-import" {
			continue
		}
		if err := pruneTestsByImports(root, module); err != nil {
			return err
		}
	}
	return nil
}

func pruneTestsByImports(root, module string) error {
	var testFiles []string
	packages := map[string]bool{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			packages[filepath.ToSlash(filepath.Dir(rel))] = true
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			testFiles = append(testFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}
	prefix := module + "/"
	for _, path := range testFiles {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse test %s: %w", path, err)
		}
		missing := false
		for _, importSpec := range file.Imports {
			name := strings.Trim(importSpec.Path.Value, `"`)
			rel, ok := strings.CutPrefix(name, prefix)
			if ok && !packages[rel] {
				missing = true
				break
			}
		}
		if missing {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove unsatisfied test %s: %w", path, err)
			}
		}
	}
	return nil
}
