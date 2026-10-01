package render

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// narrowDrivers keeps the generated Descriptor.Drivers declaration aligned with
// the driver directories selected by the manifest.
func narrowDrivers(root string, selected map[string][]string) error {
	for name, drivers := range selected {
		if len(drivers) == 0 {
			continue
		}
		dir := filepath.Join(root, "internal", "capabilities", name)
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("selected capability %s: %w", name, err)
		}
		found := false
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			updated, changed, err := patchDescriptorDrivers(content, drivers)
			if err != nil {
				return fmt.Errorf("patch drivers in %s: %w", filepath.ToSlash(filepath.Join("internal/capabilities", name, entry.Name())), err)
			}
			if !changed {
				return nil
			}
			if err := os.WriteFile(path, updated, 0o644); err != nil {
				return err
			}
			found = true
			return nil
		})
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("selected capability %s has no Descriptor.Drivers declaration", name)
		}
	}
	return nil
}

func patchDescriptorDrivers(content []byte, drivers []string) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "descriptor.go", content, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	changed := false
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, ident := range value.Names {
				if ident.Name != "Descriptor" || i >= len(value.Values) {
					continue
				}
				literal, ok := value.Values[i].(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, element := range literal.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					key, keyOK := field.Key.(*ast.Ident)
					if !ok || !keyOK || key.Name != "Drivers" {
						continue
					}
					field.Value = driverLiteral(drivers)
					changed = true
				}
			}
		}
		return true
	})
	if !changed {
		return content, false, nil
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return nil, false, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, false, err
	}
	return formatted, true, nil
}

func driverLiteral(drivers []string) ast.Expr {
	values := slices.Clone(drivers)
	slices.Sort(values)
	literal := &ast.CompositeLit{Type: &ast.ArrayType{Elt: &ast.Ident{Name: "string"}}}
	for _, driver := range values {
		literal.Elts = append(literal.Elts, &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", driver)})
	}
	return literal
}
