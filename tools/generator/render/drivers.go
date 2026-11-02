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
	"strconv"
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
	var inspectErr error
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
					if !ok {
						continue
					}
					key, keyOK := field.Key.(*ast.Ident)
					if !keyOK {
						continue
					}
					switch key.Name {
					case "Drivers":
						field.Value = driverLiteral(drivers)
						changed = true
					case "DriverSpecs":
						filtered, filteredChanged, err := filterDriverSpecs(field.Value, drivers)
						if err != nil {
							inspectErr = err
							return false
						}
						if filteredChanged {
							field.Value = filtered
							changed = true
						}
					}
				}
			}
		}
		return true
	})
	if inspectErr != nil {
		return nil, false, inspectErr
	}
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

func filterDriverSpecs(expr ast.Expr, drivers []string) (ast.Expr, bool, error) {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return expr, false, fmt.Errorf("DriverSpecs must be a composite literal")
	}
	selected := make(map[string]bool, len(drivers))
	for _, driver := range drivers {
		selected[driver] = true
	}
	kept := make([]ast.Expr, 0, len(literal.Elts))
	changed := false
	for _, element := range literal.Elts {
		name, ok := driverSpecName(element)
		if !ok {
			return expr, false, fmt.Errorf("DriverSpecs element must be a composite literal with a string Name")
		}
		if selected[name] {
			kept = append(kept, element)
			continue
		}
		changed = true
	}
	if !changed {
		return expr, false, nil
	}
	literal.Elts = kept
	return literal, true, nil
}

func driverSpecName(element ast.Expr) (string, bool) {
	spec, ok := element.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	for _, field := range spec.Elts {
		keyValue, ok := field.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok || key.Name != "Name" {
			continue
		}
		value, ok := keyValue.Value.(*ast.BasicLit)
		if !ok || value.Kind != token.STRING {
			return "", false
		}
		name, err := strconv.Unquote(value.Value)
		if err != nil {
			return "", false
		}
		return name, true
	}
	return "", false
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
