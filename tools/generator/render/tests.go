package render

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"jimu/tools/generator/manifest"
)

type testDeps struct {
	pkg            string
	missingImports []string
	missingAssets  []string
	compositionDep bool
	decls          map[string]bool
	refs           map[string]bool
}

func (d testDeps) satisfied(catalogComplete bool) bool {
	return len(d.missingImports) == 0 && len(d.missingAssets) == 0 &&
		(catalogComplete || !d.compositionDep)
}

func pruneTests(root, module string, rules []manifest.PruneRule) error {
	for _, rule := range rules {
		if rule.Kind != "test-import" {
			continue
		}
		if err := pruneTestsByImports(root, module, rule.Reason == "catalog-complete"); err != nil {
			return err
		}
	}
	return nil
}

func pruneTestsByImports(root, module string, catalogComplete bool) error {
	pkgs := map[string]bool{}
	var testRels []string
	if err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			testRels = append(testRels, filepath.ToSlash(rel))
			return nil
		}
		pkgs[filepath.ToSlash(filepath.Dir(rel))] = true
		return nil
	}); err != nil {
		return err
	}
	slices.Sort(testRels)

	prefix := module + "/"
	absent := missingAssetPrefixes(root)
	deps := make(map[string]testDeps, len(testRels))
	type groupKey struct{ dir, pkg string }
	groups := map[groupKey][]string{}
	for _, rel := range testRels {
		dep, err := analyzeTestFile(filepath.Join(root, filepath.FromSlash(rel)), prefix, pkgs, absent)
		if err != nil {
			return err
		}
		deps[rel] = dep
		key := groupKey{dir: filepath.ToSlash(filepath.Dir(rel)), pkg: dep.pkg}
		groups[key] = append(groups[key], rel)
	}

	drop := map[string]bool{}
	for _, files := range groups {
		var unsatisfied []string
		droppedDecls := map[string]bool{}
		for _, rel := range files {
			if deps[rel].satisfied(catalogComplete) {
				continue
			}
			unsatisfied = append(unsatisfied, rel)
			for name := range deps[rel].decls {
				droppedDecls[name] = true
			}
		}
		if len(unsatisfied) == 0 {
			continue
		}
		cascade := false
		for _, rel := range files {
			if !deps[rel].satisfied(catalogComplete) {
				continue
			}
			for name := range deps[rel].refs {
				if droppedDecls[name] {
					cascade = true
					break
				}
			}
			if cascade {
				break
			}
		}
		if cascade {
			for _, rel := range files {
				drop[rel] = true
			}
			continue
		}
		for _, rel := range unsatisfied {
			drop[rel] = true
		}
	}

	values := make([]string, 0, len(drop))
	for rel := range drop {
		values = append(values, rel)
	}
	slices.Sort(values)
	for _, rel := range values {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("remove unsatisfiable test %s: %w", rel, err)
		}
	}
	return assertNoProductionGap(root, prefix, pkgs)
}

func analyzeTestFile(file, prefix string, pkgs map[string]bool, absent []string) (testDeps, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return testDeps{}, fmt.Errorf("parse %s: %w", filepath.ToSlash(file), err)
	}
	deps := testDeps{pkg: parsed.Name.Name, decls: map[string]bool{}, refs: map[string]bool{}}
	for _, spec := range parsed.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		if name == prefix+"internal/capabilities/catalog" {
			deps.compositionDep = true
			continue
		}
		rel, ok := strings.CutPrefix(name, prefix)
		if ok && !pkgs[rel] {
			deps.missingImports = append(deps.missingImports, name)
		}
	}
	deps.missingAssets = assetRefsOf(parsed, absent)
	for _, decl := range parsed.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Recv == nil {
				recordDecl(&deps, value.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				switch item := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range item.Names {
						recordDecl(&deps, name.Name)
					}
				case *ast.TypeSpec:
					recordDecl(&deps, item.Name.Name)
				}
			}
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name != "_" {
			deps.refs[ident.Name] = true
		}
		return true
	})
	return deps, nil
}

func recordDecl(deps *testDeps, name string) {
	if name != "_" {
		deps.decls[name] = true
	}
}

func assertNoProductionGap(root, prefix string, pkgs map[string]bool) error {
	return filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		missing, err := missingModuleImports(file, prefix, pkgs)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return fmt.Errorf("generated production file %s imports missing packages %s", filepath.ToSlash(file), strings.Join(missing, ", "))
		}
		return nil
	})
}

func missingModuleImports(file, prefix string, pkgs map[string]bool) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.ToSlash(file), err)
	}
	var missing []string
	for _, spec := range parsed.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		rel, ok := strings.CutPrefix(name, prefix)
		if ok && !pkgs[rel] {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

func missingAssetPrefixes(root string) []string {
	var missing []string
	if _, err := os.Stat(filepath.Join(root, "docs", "openapi")); os.IsNotExist(err) {
		missing = append(missing, "docs/openapi")
	}
	return missing
}

func assetRefsOf(parsed *ast.File, absent []string) []string {
	if len(absent) == 0 {
		return nil
	}
	found := map[string]bool{}
	record := func(literal string) {
		clean := cleanAssetLiteral(literal)
		if clean == "" {
			return
		}
		for _, prefix := range absent {
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") ||
				strings.HasSuffix(clean, "/"+prefix) || strings.Contains(clean, "/"+prefix+"/") {
				found[prefix] = true
			}
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.BasicLit:
			if value.Kind == token.STRING {
				if text, err := strconv.Unquote(value.Value); err == nil {
					record(text)
				}
			}
		case *ast.CallExpr:
			parts := make([]string, 0, len(value.Args))
			for _, arg := range value.Args {
				literal, ok := arg.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					parts = nil
					break
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					parts = nil
					break
				}
				parts = append(parts, text)
			}
			if len(parts) > 0 {
				record(strings.Join(parts, "/"))
			}
		}
		return true
	})
	values := make([]string, 0, len(found))
	for value := range found {
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}

func cleanAssetLiteral(literal string) string {
	value := strings.TrimSpace(filepath.ToSlash(literal))
	for strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		if strings.HasPrefix(value, "./") {
			value = value[2:]
		} else {
			value = value[3:]
		}
	}
	return path.Clean(value)
}
