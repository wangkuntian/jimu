package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
)

// checkCapabilityStructure verifies the two assembly entrypoints in each capability root package.
func checkCapabilityStructure(root string, names []string) error {
	modulePath, err := readModulePath(root)
	if err != nil {
		return err
	}
	for _, name := range slices.Sorted(slices.Values(names)) {
		dir := filepath.Join(root, "internal", "capabilities", name)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("capability %q at %s: %w", name, dir, err)
		}
		descriptors, wires := 0, 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return fmt.Errorf("capability %q at %s: %w", name, path, err)
			}
			imports := map[string]string{}
			for _, imp := range file.Imports {
				pkgPath, _ := strconv.Unquote(imp.Path.Value)
				alias := filepath.Base(pkgPath)
				if imp.Name != nil {
					alias = imp.Name.Name
				}
				imports[alias] = pkgPath
			}
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					if d.Tok != token.VAR {
						continue
					}
					for _, spec := range d.Specs {
						v, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, id := range v.Names {
							if id.Name == "Descriptor" {
								descriptors++
								if !descriptorType(v, imports, modulePath) {
									return fmt.Errorf("capability %q at %s: Descriptor must be contract.Descriptor", name, path)
								}
							}
						}
					}
				case *ast.FuncDecl:
					if d.Name.Name == "Wire" && d.Recv == nil {
						wires++
						if !validWireSignature(d.Type, imports, modulePath) {
							return fmt.Errorf("capability %q at %s: Wire must have signature func(*assembly.Context) (contract.Module, error)", name, path)
						}
					}
				}
			}
		}
		if descriptors != 1 || wires != 1 {
			return fmt.Errorf("capability %q at %s: expected one Descriptor and one Wire, got %d and %d", name, dir, descriptors, wires)
		}
	}
	return nil
}

func readModulePath(root string) (string, error) {
	modFile, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read module path: %w", err)
	}
	modulePath := modfile.ModulePath(modFile)
	if modulePath == "" {
		return "", fmt.Errorf("read module path: go.mod has no module directive")
	}
	return modulePath, nil
}

func descriptorType(v *ast.ValueSpec, imports map[string]string, modulePath string) bool {
	if qualifiedType(v.Type, imports, modulePath+"/internal/contract", "Descriptor") {
		return true
	}
	return len(v.Values) == 1 && qualifiedType(compositeType(v.Values[0]), imports, modulePath+"/internal/contract", "Descriptor")
}

func compositeType(expr ast.Expr) ast.Expr {
	if lit, ok := expr.(*ast.CompositeLit); ok {
		return lit.Type
	}
	return nil
}

func qualifiedType(expr ast.Expr, imports map[string]string, pkgPath, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && imports[id.Name] == pkgPath
}

func validWireSignature(fn *ast.FuncType, imports map[string]string, modulePath string) bool {
	if fn.TypeParams != nil || fn.Params == nil || len(fn.Params.List) != 1 || fn.Results == nil || len(fn.Results.List) != 2 {
		return false
	}
	pointer, ok := fn.Params.List[0].Type.(*ast.StarExpr)
	if !ok || len(fn.Params.List[0].Names) > 1 || !qualifiedType(pointer.X, imports, modulePath+"/internal/assembly", "Context") {
		return false
	}
	if !qualifiedType(fn.Results.List[0].Type, imports, modulePath+"/internal/contract", "Module") {
		return false
	}
	errType, ok := fn.Results.List[1].Type.(*ast.Ident)
	return ok && errType.Name == "error"
}
