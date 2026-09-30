package render

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type cliImport struct {
	Alias string
	Path  string
}

func renderCLI(sourceRoot, destinationRoot, source, destination string, data any) error {
	sourcePath, err := safeJoin(sourceRoot, source)
	if err != nil {
		return err
	}
	destinationPath, err := safeJoin(destinationRoot, destination)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read framework %s: %w", source, err)
	}
	imports, err := cliImportsFromData(data)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, source, strings.NewReader(string(content)), parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse framework %s: %w", source, err)
	}
	offsets := lineOffsets(string(content))
	lineOf := func(pos token.Pos) int { return fset.PositionFor(pos, false).Line }
	lineStart := func(line int) int { return offsets[line-1] }
	lineEnd := func(line int) int {
		if line >= len(offsets) {
			return len(content)
		}
		return offsets[line]
	}
	dropLines := func(from, to int) textEdit {
		return textEdit{start: lineStart(from), end: lineEnd(to)}
	}

	var edits []textEdit
	lastFrameworkImport := 0
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if path == "jimu/tools/generator" || isCapabilityCLIImportPath(path) {
			edits = append(edits, dropLines(lineOf(spec.Pos()), lineOf(spec.End())))
			continue
		}
		if strings.HasPrefix(path, "jimu/") && lineOf(spec.End()) > lastFrameworkImport {
			lastFrameworkImport = lineOf(spec.End())
		}
	}

	insertAt := -1
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "init" || fn.Body == nil {
			continue
		}
		insertAt = lineStart(lineOf(fn.Body.Rbrace))
		for _, stmt := range fn.Body.List {
			if !cliInitStmtDropped(stmt) {
				continue
			}
			from := lineOf(stmt.Pos())
			if comment := commentGroupEndingAtLine(fset, file, from-1); comment != nil {
				from = lineOf(comment.Pos())
			}
			edits = append(edits, dropLines(from, lineOf(stmt.End())))
		}
	}

	if len(imports) > 0 {
		if lastFrameworkImport == 0 {
			return fmt.Errorf("render %s: no framework import found to anchor capability cli imports", source)
		}
		var importText strings.Builder
		for _, item := range imports {
			fmt.Fprintf(&importText, "\t%s %q\n", item.Alias, item.Path)
		}
		at := lineEnd(lastFrameworkImport)
		edits = append(edits, textEdit{start: at, end: at, text: importText.String()})
		if insertAt < 0 {
			return fmt.Errorf("render %s: no init() to register capability commands", source)
		}
		var registrationText strings.Builder
		registrationText.WriteString("\t// 能力自带的命令（manifest selection）：只注册已选中能力自带的命令。\n")
		for _, item := range imports {
			fmt.Fprintf(&registrationText, "\trootCmd.AddCommand(%s.Commands()...)\n", item.Alias)
		}
		edits = append(edits, textEdit{start: insertAt, end: insertAt, text: registrationText.String()})
	}

	result := generatedHeader + cliMainHeader + applyTextEdits(string(content), edits)
	formatted, err := format.Source([]byte(result))
	if err != nil {
		return fmt.Errorf("format rendered %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", destination, err)
	}
	if err := os.WriteFile(destinationPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", destination, err)
	}
	return nil
}

func cliImportsFromData(data any) ([]cliImport, error) {
	value, ok := data.(map[string]any)
	if !ok {
		if data == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("cli template data must be an object")
	}
	raw, ok := value["CLIImports"]
	if !ok || raw == nil {
		return nil, nil
	}
	items, ok := raw.([]map[string]any)
	if ok {
		return cliImportsFromMaps(items)
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("cli template data CLIImports must be an array")
	}
	imports := make([]cliImport, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cli template data CLIImports item must be an object")
		}
		alias, _ := item["Alias"].(string)
		path, _ := item["Path"].(string)
		if alias == "" || path == "" {
			return nil, fmt.Errorf("cli template data CLIImports item requires Alias and Path")
		}
		imports = append(imports, cliImport{Alias: alias, Path: path})
	}
	return imports, nil
}

func cliImportsFromMaps(values []map[string]any) ([]cliImport, error) {
	imports := make([]cliImport, 0, len(values))
	for _, value := range values {
		alias, _ := value["Alias"].(string)
		path, _ := value["Path"].(string)
		if alias == "" || path == "" {
			return nil, fmt.Errorf("cli template data CLIImports item requires Alias and Path")
		}
		imports = append(imports, cliImport{Alias: alias, Path: path})
	}
	return imports, nil
}

func isCapabilityCLIImportPath(path string) bool {
	rest, ok := strings.CutPrefix(path, "jimu/"+"internal/"+"capabilities/")
	return ok && strings.Count(rest, "/") == 1 && strings.HasSuffix(rest, "/cli")
}

func cliInitStmtDropped(stmt ast.Stmt) bool {
	if stmtReferencesAny(stmt, "newCmd", "capabilityCmd") {
		return true
	}
	dropped := false
	ast.Inspect(stmt, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "Commands" {
			dropped = true
		}
		return true
	})
	return dropped
}

func stmtReferencesAny(stmt ast.Stmt, names ...string) bool {
	found := false
	ast.Inspect(stmt, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && slices.Contains(names, ident.Name) {
			found = true
		}
		return true
	})
	return found
}

func commentGroupEndingAtLine(fset *token.FileSet, file *ast.File, line int) *ast.CommentGroup {
	for _, group := range file.Comments {
		if fset.PositionFor(group.End(), false).Line == line {
			return group
		}
	}
	return nil
}

type textEdit struct {
	start int
	end   int
	text  string
}

func applyTextEdits(source string, edits []textEdit) string {
	slices.SortFunc(edits, func(a, b textEdit) int { return b.start - a.start })
	for _, edit := range edits {
		source = source[:edit.start] + edit.text + source[edit.end:]
	}
	return source
}

func lineOffsets(source string) []int {
	offsets := []int{0}
	for index := 0; index < len(source); index++ {
		if source[index] == '\n' {
			offsets = append(offsets, index+1)
		}
	}
	offsets = append(offsets, len(source))
	return offsets
}

const generatedHeader = "// Code generated by jimu new; DO NOT EDIT.\n"

const cliMainHeader = `// 本文件由生成器按 manifest selection 裁剪：框架脚手架命令（jimu new / jimu capability create / jimu capability add）是**框架仓**的职责，
// 生成项目不含 tools/generator，也不注册这些命令；能力自带命令只注册已选中的能力。
`
