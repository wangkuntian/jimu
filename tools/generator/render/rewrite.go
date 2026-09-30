package render

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"jimu/tools/generator/manifest"
)

func applyRewrite(root, module string, action manifest.RewriteAction) error {
	files := action.Files
	if len(files) == 0 {
		files = allFiles(root)
	}
	for _, rel := range files {
		path, err := safeJoin(root, rel)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read rewrite target %s: %w", rel, err)
		}
		if !utf8.Valid(content) {
			continue
		}
		from, to := action.From, action.To
		if action.Kind == "module" {
			if module != "" {
				to = module
			}
			updated, changed, err := rewriteModuleFile(filepath.Base(path), string(content), from, to)
			if err != nil {
				return fmt.Errorf("rewrite %s: %w", rel, err)
			}
			if changed {
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
					return err
				}
			}
			continue
		}
		updated := strings.ReplaceAll(string(content), from, to)
		if updated != string(content) {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

func allFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil && entry.Type()&fs.ModeSymlink == 0 {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func rewriteModuleFile(base, content, from, module string) (string, bool, error) {
	if from == "" || module == "" {
		return content, false, fmt.Errorf("source and destination module paths are required")
	}
	if base == "go.mod" {
		updated, found := rewriteGoModDirective(content, from, module)
		if !found && from != module {
			return content, false, fmt.Errorf("go.mod module directive %q was not found", from)
		}
		return updated, updated != content, nil
	}
	if strings.HasSuffix(base, ".go") {
		updated, err := rewriteGoSource(content, from, module)
		return updated, updated != content, err
	}
	if !isTextFile(base) {
		return content, false, nil
	}
	updated := strings.ReplaceAll(content, from+"/", module+"/")
	updated = strings.ReplaceAll(updated, "module="+from, "module="+module)
	updated = strings.ReplaceAll(updated, "$(BIN_DIR)/jimu-", "$(BIN_DIR)/"+artifactName(module)+"-")
	return updated, updated != content, nil
}

func rewriteGoModDirective(content, from, module string) (string, bool) {
	lines := strings.SplitAfter(content, "\n")
	found := false
	for index, line := range lines {
		body, eol := line, ""
		if strings.HasSuffix(body, "\n") {
			body = strings.TrimSuffix(body, "\n")
			eol = "\n"
		}
		trimmed := strings.TrimSpace(body)
		if !strings.HasPrefix(trimmed, "module ") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) < 2 || parts[1] != from {
			continue
		}
		lines[index] = "module " + module + eol
		found = true
	}
	return strings.Join(lines, ""), found
}

func rewriteGoSource(source, from, module string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "rewrite.go", strings.NewReader(source), parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse go source: %w", err)
	}
	offsetOf := func(pos token.Pos) int { return fset.PositionFor(pos, false).Offset }
	var edits []textEdit
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		rest, ok := strings.CutPrefix(value, from+"/")
		if !ok {
			continue
		}
		edits = append(edits, textEdit{start: offsetOf(spec.Path.Pos()), end: offsetOf(spec.Path.End()), text: strconv.Quote(module + "/" + rest)})
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok.String() != "const" {
			continue
		}
		for _, spec := range gen.Specs {
			values, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range values.Names {
				if name.Name != "modulePath" || index >= len(values.Values) {
					continue
				}
				literal, ok := values.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind.String() != "STRING" {
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil || value != from {
					continue
				}
				edits = append(edits, textEdit{start: offsetOf(literal.Pos()), end: offsetOf(literal.End()), text: strconv.Quote(module)})
			}
		}
	}
	if len(edits) == 0 {
		return source, nil
	}
	formatted, err := format.Source([]byte(applyTextEdits(source, edits)))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func isTextFile(base string) bool {
	if base == "Makefile" || base == "Dockerfile" {
		return true
	}
	switch filepath.Ext(base) {
	case ".go", ".mod", ".sum", ".sh", ".yaml", ".yml", ".mk", ".md", ".conf", ".json", ".tmpl", ".proto":
		return true
	default:
		return false
	}
}

func artifactName(module string) string {
	base := filepath.Base(module)
	var builder strings.Builder
	for _, value := range strings.ToLower(base) {
		if (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '-' {
			builder.WriteRune(value)
		} else {
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}
