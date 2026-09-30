package generator

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

// 生成项目的 cmd/cli/main.go 由本文件渲染（单形态裁剪版）。文本手术而非整份模板：
// 内核 CLI 的 migrate/seed/config/version 命令树很长且会持续演进，逐行复刻必然漂移；
// 这里以框架仓的 cmd/cli/main.go 为唯一来源，只做三处**受控**裁剪/注入：
//
//	① 去掉 tools/generator import（脚手架是框架仓的职责，
//	   生成项目不含 tools/generator）；
//	② 去掉框架脚手架命令（new / capability create / capability add）的注册；
//	③ 按选中能力注入 capabilities/<cap>/cli 的 import 与命令注册（P2.6 接缝）。
const cliMainRel = "cmd/cli/main.go"

// capabilityCLIImport 是生成项目要 import 的能力自带命令包。
type capabilityCLIImport struct {
	Alias string
	Path  string
}

// capabilityCLIImports 返回**选中能力**里自带 CLI 命令的那些（按声明顺序）：
// 只有框架仓真实存在 internal/capabilities/<cap>/cli 目录的能力才算。
func capabilityCLIImports(root string, set CapabilitySet) []capabilityCLIImport {
	var out []capabilityCLIImport
	for _, name := range set.Declared {
		dir := filepath.Join(root, filepath.FromSlash(capabilityDirPrefix), name, "cli")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			out = append(out, capabilityCLIImport{
				Alias: name + "cli",
				Path:  frameworkModule + "/" + capabilityDirPrefix + "/" + name + "/cli",
			})
		}
	}
	return out
}

// isCapabilityCLIImportPath 判定 import 路径是否指向某个能力的 cli 包。
func isCapabilityCLIImportPath(path string) bool {
	rest, ok := strings.CutPrefix(path, frameworkModule+"/"+capabilityDirPrefix+"/")
	if !ok {
		return false
	}
	return strings.Count(rest, "/") == 1 && strings.HasSuffix(rest, "/cli")
}

// RenderCLIMain 渲染生成项目的 cmd/cli/main.go。必须在 RewriteModule 之前调用（写入的
// import 仍是框架模块前缀）。
func RenderCLIMain(root, dst string, set CapabilitySet) error {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(cliMainRel)))
	if err != nil {
		return fmt.Errorf("read framework %s: %w", cliMainRel, err)
	}
	src := string(content)
	fset := token.NewFileSet()
	// io.Reader 传源码：go/parser 的 string 形式必须以换行开头才当源码（见 astutil.go 注）。
	file, err := parser.ParseFile(fset, baseName("cli_main"), strings.NewReader(src), parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse framework %s: %w", cliMainRel, err)
	}
	off := lineOffsets(src)
	lineOf := func(p token.Pos) int { return fset.PositionFor(p, false).Line }
	lineStart := func(n int) int { return off[n-1] }
	lineEnd := func(n int) int {
		if n >= len(off) {
			return len(src)
		}
		return off[n]
	}
	dropLines := func(from, to int) textEdit {
		return textEdit{start: lineStart(from), end: lineEnd(to)}
	}

	caps := capabilityCLIImports(root, set)
	var edits []textEdit

	// ① import：去掉脚手架与**当前**写死的能力 cli 包，记住最后一个框架自身 import 的行号。
	lastFrameworkImport := 0
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if path == frameworkModule+"/tools/generator" || isCapabilityCLIImportPath(path) {
			edits = append(edits, dropLines(lineOf(spec.Pos()), lineOf(spec.End())))
			continue
		}
		if strings.HasPrefix(path, frameworkModule+"/") && lineOf(spec.End()) > lastFrameworkImport {
			lastFrameworkImport = lineOf(spec.End())
		}
	}

	// ① init：去掉 new/capability 的注册与写死的能力命令注册，并记住注入点。
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
			if c := commentGroupEndingAtLine(fset, file, from-1); c != nil {
				from = lineOf(c.Pos())
			}
			edits = append(edits, dropLines(from, lineOf(stmt.End())))
		}
	}

	// ③ 注入选中能力的 import 与命令注册。
	if len(caps) > 0 {
		if lastFrameworkImport == 0 {
			return fmt.Errorf("render %s: no framework import found to anchor capability cli imports", cliMainRel)
		}
		var imports strings.Builder
		for _, c := range caps {
			fmt.Fprintf(&imports, "\t%s %q\n", c.Alias, c.Path)
		}
		at := lineEnd(lastFrameworkImport)
		edits = append(edits, textEdit{start: at, end: at, text: imports.String()})

		if insertAt < 0 {
			return fmt.Errorf("render %s: no init() to register capability commands", cliMainRel)
		}
		var regs strings.Builder
		regs.WriteString("\t// 能力自带的命令（P2.6 接缝）：只注册已选中能力自带的命令。\n")
		for _, c := range caps {
			fmt.Fprintf(&regs, "\trootCmd.AddCommand(%s.Commands()...)\n", c.Alias)
		}
		edits = append(edits, textEdit{start: insertAt, end: insertAt, text: regs.String()})
	}

	out := generatedHeader + cliMainHeader + applyTextEdits(src, edits)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return fmt.Errorf("format rendered %s: %w", cliMainRel, err)
	}
	target := filepath.Join(dst, filepath.FromSlash(cliMainRel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", cliMainRel, err)
	}
	if err := os.WriteFile(target, formatted, goFileMode); err != nil {
		return fmt.Errorf("write %s: %w", cliMainRel, err)
	}
	return nil
}

// cliMainHeader 说明本文件的裁剪口径（放在 package 之前，即包文档）。
const cliMainHeader = `// 本文件由生成器按选中能力裁剪：框架脚手架命令（jimu new / jimu capability create / jimu capability add）是**框架仓**的职责，
// 生成项目不含 tools/generator，也不注册这些命令；能力自带命令只注册已选中的能力。
`

// textEdit 是一次源码字节区间替换（start == end 表示插入）。
type textEdit struct {
	start, end int
	text       string
}

// applyTextEdits 从后往前应用编辑（前面的偏移不受后面编辑影响）。
func applyTextEdits(src string, edits []textEdit) string {
	slices.SortFunc(edits, func(a, b textEdit) int { return b.start - a.start })
	for _, e := range edits {
		src = src[:e.start] + e.text + src[e.end:]
	}
	return src
}

// lineOffsets 返回每行**起始**字节偏移，下标 i 对应 1-based 行号 i+1；末尾追加 len(src)，
// 于是 off[l] 恒为第 l 行之后（下一行起始）的偏移。
func lineOffsets(src string) []int {
	off := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			off = append(off, i+1)
		}
	}
	off = append(off, len(src))
	return off
}

// cliInitStmtDropped 判定 init() 里的哪条语句属于「框架脚手架」而必须从生成项目里去掉：
// 引用 newCmd/capabilityCmd 的注册（capability add 依赖 tools/generator，
// 生成项目不含该工具树，见 kernelExcludes），以及形如 `<x>cli.Commands()` 的能力命令注册
// （后者由选中集重新注入）。
func cliInitStmtDropped(stmt ast.Stmt) bool {
	if stmtReferencesAny(stmt, "newCmd", "capabilityCmd") {
		return true
	}
	dropped := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Commands" {
			dropped = true
		}
		return true
	})
	return dropped
}

// stmtReferencesAny 判定语句里是否出现指定标识符。
func stmtReferencesAny(stmt ast.Stmt, names ...string) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if slices.Contains(names, id.Name) {
			found = true
		}
		return true
	})
	return found
}

// commentGroupEndingAtLine 返回**结束于**指定行的注释组（用于把语句上方的说明注释一并删掉）。
func commentGroupEndingAtLine(fset *token.FileSet, file *ast.File, line int) *ast.CommentGroup {
	for _, g := range file.Comments {
		if fset.PositionFor(g.End(), false).Line == line {
			return g
		}
	}
	return nil
}
