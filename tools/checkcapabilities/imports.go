package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// checkCapabilityImports rejects direct imports from one capability into another.
// catalog is the only composition root inside internal/capabilities.
func checkCapabilityImports(root string) error {
	modulePath, err := readModulePath(root)
	if err != nil {
		return err
	}
	capRoot := filepath.Join(root, "internal", "capabilities")
	importPrefix := modulePath + "/internal/capabilities/"
	return filepath.WalkDir(capRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(capRoot, path)
		if err != nil {
			return err
		}
		source := strings.Split(filepath.ToSlash(rel), "/")[0]
		if source == "catalog" {
			if entry.IsDir() && rel == "catalog" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse capability imports %s: %w", path, err)
		}
		for _, imp := range parsed.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return fmt.Errorf("parse capability import %s: %w", path, err)
			}
			if !strings.HasPrefix(importPath, importPrefix) {
				continue
			}
			target := strings.Split(strings.TrimPrefix(importPath, importPrefix), "/")[0]
			if target != "" && target != source {
				return fmt.Errorf("capability %q at %s imports %s (capability %q)", source, path, importPath, target)
			}
		}
		return nil
	})
}
