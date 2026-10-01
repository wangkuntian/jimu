package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// checkGeneratorBoundary keeps the manifest execution packages independent from
// framework internals. The root generator package is the public facade and
// frameworkmanifest is the explicit adapter; the remaining subpackages are
// scanned as generator core.
func checkGeneratorBoundary(root string) error {
	modulePath, err := readModulePath(root)
	if err != nil {
		return err
	}
	generatorRoot := filepath.Join(root, "tools", "generator")
	if _, err := os.Stat(generatorRoot); err != nil {
		if os.IsNotExist(err) {
			return nil // generated projects intentionally do not copy the generator
		}
		return fmt.Errorf("stat generator root: %w", err)
	}
	boundaryPrefixes := []string{
		modulePath + "/internal/capabilities",
		modulePath + "/internal/profiles",
		modulePath + "/internal/contract",
	}
	var violations []string
	err = filepath.WalkDir(generatorRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(generatorRoot, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if entry.IsDir() {
			if relSlash == "testdata" || strings.HasPrefix(relSlash, "testdata/") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || !isGeneratorCorePath(relSlash) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse generator boundary %s: %w", path, err)
		}
		for _, imp := range file.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return fmt.Errorf("parse generator import %s: %w", path, err)
			}
			for _, prefix := range boundaryPrefixes {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					violations = append(violations, fmt.Sprintf("%s imports %s", relSlash, importPath))
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}
	slices.Sort(violations)
	return fmt.Errorf("generator boundary violation:\n  %s", strings.Join(violations, "\n  "))
}

func isGeneratorCorePath(rel string) bool {
	parts := strings.Split(rel, "/")
	return len(parts) > 1 && parts[0] != "frameworkmanifest"
}
