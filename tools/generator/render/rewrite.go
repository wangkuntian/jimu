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
			if action.Kind == "deploy" && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read rewrite target %s: %w", rel, err)
		}
		if !utf8.Valid(content) {
			continue
		}
		from, to := action.From, action.To
		if action.Kind == "file-patch" {
			updated := strings.ReplaceAll(string(content), from, to)
			if updated == string(content) {
				return fmt.Errorf("rewrite target %s: patch source %q was not found", rel, from)
			}
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
				return err
			}
			continue
		}
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
		if action.Kind == "deploy" {
			updated := rewriteDeployIdentifiers(string(content), to)
			if updated != string(content) {
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
	// 镜像仓库名与镜像内 CLI 产物名沿用生成项目的 artifact 名，
	// 避免生成项目仍以框架名发布镜像（k8s 里两个项目会撞名）。
	if base == "Makefile" {
		updated = strings.ReplaceAll(updated, "IMAGE_REPO ?= jimu", "IMAGE_REPO ?= "+artifactName(module))
	}
	if base == "Dockerfile" {
		updated = strings.ReplaceAll(updated, "-o jimu ./cmd/cli", "-o "+artifactName(module)+" ./cmd/cli")
		updated = strings.ReplaceAll(updated, "/app/jimu .", "/app/"+artifactName(module)+" .")
	}
	return updated, updated != content, nil
}

// rewriteDeployIdentifiers 把部署清单里的框架名换成生成项目的 artifact 名。
// 只替换「项目标识」的位置：namespace、资源名与 app 标签、镜像仓库、secret/config 名。
// 指标名、OpenObserve stream 名与告警规则的 jimu_ 前缀属于运行期协议，保持不动。
func rewriteDeployIdentifiers(content, name string) string {
	replacements := [][2]string{
		{"name: jimu\n", "name: " + name + "\n"},
		{"repository: jimu\n", "repository: " + name + "\n"},
		{"repository: jimu-backup", "repository: " + name + "-backup"},
		{"prefix: jimu", "prefix: " + name},
		{"serviceName: jimu", "serviceName: " + name},
		{"user: jimu\n", "user: " + name + "\n"},
		{"mysqlUser: jimu", "mysqlUser: " + name},
		{"namespace: jimu", "namespace: " + name},
		{"app: jimu-server", "app: " + name + "-server"},
		{"name: jimu-server", "name: " + name + "-server"},
		{"app: jimu-db-backup", "app: " + name + "-db-backup"},
		{"name: jimu-db-backup", "name: " + name + "-db-backup"},
		{"app: jimu-config", "app: " + name + "-config"},
		{"name: jimu-config", "name: " + name + "-config"},
		{"name: jimu-secrets", "name: " + name + "-secrets"},
		{"secretName: jimu-tls", "secretName: " + name + "-tls"},
		{"registry.example.com/jimu:", "registry.example.com/" + name + ":"},
		{"registry.example.com/jimu-backup:", "registry.example.com/" + name + "-backup:"},
		{"mariadb.jimu.svc.cluster.local", "mariadb." + name + ".svc.cluster.local"},
		{"redis.jimu.svc.cluster.local", "redis." + name + ".svc.cluster.local"},
		{"registry.example.com/jimu:1.0.0", "registry.example.com/" + name + ":1.0.0"},
	}
	for _, pair := range replacements {
		content = strings.ReplaceAll(content, pair[0], pair[1])
	}
	return content
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
