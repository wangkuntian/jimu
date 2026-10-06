package support

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// rewriteRules 是模块路径/产物名的**受控重写**规则（按顺序应用，前一条的输出是后一条的输入）。
// 只做字面量替换，不做 AST 改写：可单测、对注释与脚本同样生效。
//
// 幂等契约（精确，Fix round 3 裁定）：
//
//	(a) 被放行的 `to`（`modulePathCollides` 只拒绝 `to` 自身仍含源前缀 `jimu/` 的情形）保证
//	    **go.mod 的 module 指令与 `.go` import 前缀**的替换幂等——第二遍不再改动它们；
//	(b) **`jimu-*` 产物名规则不保证对同一棵已重写树幂等**：`$(BIN_DIR)/jimu-` 以产物名替换，
//	    若产物名自身以 `jimu-` 开头（如 `--module=example.com/jimu-app`），第二遍会把
//	    `$(BIN_DIR)/jimu-app-server` 再改成 `$(BIN_DIR)/jimu-app-app-server`。
//
// (b) 为何安全：单遍产物**正确**（名为 `jimu-app` 的项目，其产物名本就该是 `jimu-app-server`），
// 而重写只对「从框架仓新复制出来的树」执行一次——T2/T8 的调用点是临时目录语义，`--force` 也是
// 「重新复制 + 重写一次」，不会对已重写过的树二次重写。因此不把这类 `to` 判为非法（可用性）。
var rewriteRules = []struct {
	literal string // 匹配的字面量
	scope   string // "all" 全部文本文件 | "go" 仅 .go | "gomod" 仅 go.mod
	repl    func(module, name string) string
}{
	{"module jimu\n", "gomod", func(m, _ string) string { return "module " + m + "\n" }},
	{"jimu/", "all", func(m, _ string) string { return m + "/" }},
	// tools/{composereport,checkcapabilities} 的 const modulePath = "jimu"（无斜杠，靠 jimu/ 规则
	// 抓不到）；漏掉它会让生成项目的度量与门禁按 jimu/... 前缀统计，恒为空。
	{`const modulePath = "jimu"`, "go", func(m, _ string) string { return `const modulePath = "` + m + `"` }},
	// Makefile 的 protoc 选项与产物名。
	{"module=jimu", "all", func(m, _ string) string { return "module=" + m }},
	{"$(BIN_DIR)/jimu-", "all", func(_, n string) string { return "$(BIN_DIR)/" + n + "-" }},
}

// 明确**不改**：框架 CLI 二进制名与运行期标识 —— Dockerfile 的 `-o jimu`、`COPY --from=builder
// /app/jimu .`、`./jimu migrate`、`addgroup -S jimu`/`USER jimu`、Makefile 的 `DOCKER_IMAGE = jimu:latest`
// 与 `DOCKER_CONTAINER := jimu-server`、`deploy/**` 里的容器/镜像名。这些是**框架自己的名字**，
// 不是项目名（第 1 节裁定 3 的「不改框架 CLI 二进制名 jimu」）。

// textExtensions 是文本文件后缀白名单；Makefile/Dockerfile 无后缀，按文件名单独判定。
var textExtensions = map[string]bool{
	".go": true, ".mod": true, ".sum": true, ".sh": true, ".yaml": true, ".yml": true,
	".mk": true, ".md": true, ".conf": true, ".json": true, ".tmpl": true, ".proto": true,
}

// goModuleDirective 匹配 go.mod 的 module 指令行，容忍前导/尾随空白与行尾 `//` 注释。
// 规则表的 "module jimu\n" 字面量只覆盖「LF + 尾换行 + 无注释」这一常见形态；其余形态
// （CRLF、文件末尾无换行、尾空白、行尾注释）由本正则兜底，替换时保留注释与行尾。
var goModuleDirective = regexp.MustCompile(`^[ \t]*module[ \t]+jimu(?:[ \t]+(//.*?))?[ \t]*$`)

// goModuleValue 取 module 指令的值（module 后的第一个 token），用于写盘前的 fail-closed 校验；
// 它与 goModuleDirective 相互独立，避免「正则没匹配上 ⇒ 静默漏改」的 fail-open。
var goModuleValue = regexp.MustCompile(`(?m)^[ \t]*module[ \t]+([^ \t\r\n]+)`)

func baseName(name string) string { return "jimu_rewrite_" + name }

type textEdit struct {
	start, end int
	text       string
}

func applyTextEdits(src string, edits []textEdit) string {
	slices.SortFunc(edits, func(a, b textEdit) int { return b.start - a.start })
	for _, edit := range edits {
		src = src[:edit.start] + edit.text + src[edit.end:]
	}
	return src
}

// RewriteModule 在 root 下应用 rewriteRules：from 恒为 "jimu"，to 为模块路径，name = 产物名
// （= path.Base(module) 规范化：小写、非 [a-z0-9-] 折叠为 "-"、去首尾 "-"）。
// 文本文件判定：后缀在白名单内（.go/.mod/.sum/.sh/.yaml/.yml/.mk/.md/.conf/.json/.tmpl/.proto/
// Dockerfile/Makefile）+ 内容为合法 UTF-8；二进制文件（图片等）跳过。
// `.go` 文件改写后一律过 go/format；格式化失败即返回错误（由调用方回滚临时目录）。
// 写盘前 fail-closed 两条：`to` 自身仍含源前缀 `jimu/`（那种 `to` 单遍即坏）；go.mod 因任何原因
// 仍是 `module jimu`（宁可失败也不静默产出 `module jimu` 的项目）。调用方按原样回滚临时目录。
// 幂等范围见 rewriteRules 的 (a)/(b)：go.mod 与 `.go` import 幂等；`jimu-*` 产物名规则只保证单遍正确。
// 返回被改写的相对路径（排序）。
func RewriteModule(root, from, to string) ([]string, error) {
	// 规则表里的字面量按框架模块路径硬编码（设计上 from 恒为 "jimu"，见 rewriteRules 注释）；
	// 传入别的前缀没有可应用的规则，直接报错而不是静默不改。
	if from != "jimu" {
		return nil, fmt.Errorf("rewrite: unsupported source module path %q", from)
	}
	name := artifactName(to)
	if modulePathCollides(to, name) {
		return nil, fmt.Errorf("rewrite: module path %q collides with source prefix %q; literal rewrite cannot be idempotent", to, from)
	}
	changed := make([]string, 0, 16)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		base := d.Name()
		if !isRewriteTarget(base) {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		if !utf8.Valid(content) {
			return nil
		}
		rewritten, modified, err := rewriteFileContent(base, string(content), to, name)
		if err != nil {
			return fmt.Errorf("rewrite %s: %w", relPath(root, p), err)
		}
		if base == "go.mod" && to != from && moduleDirectiveValue(rewritten) == from {
			// 兜底正则也没命中（如 CR-only 行尾）：fail-open，必须报错而不是静默产出
			// `module jimu` 的项目。这一步与「是否有改动」无关。
			return fmt.Errorf("rewrite: go.mod module directive still %q after rewrite (%s)", from, relPath(root, p))
		}
		if !modified {
			return nil
		}
		out := []byte(rewritten)
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", relPath(root, p), err)
		}
		if err := os.WriteFile(p, out, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", relPath(root, p), err)
		}
		changed = append(changed, relPath(root, p))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(changed)
	return changed, nil
}

// rewriteFileContent 按文件类型选择重写策略（Fix round 3 / C2）：
//
//	go.mod    —— 受控字面量规则 + module 指令的宽容兜底（尾空白/行尾注释/CRLF）；
//	*.go      —— **import 感知**重写（见 rewriteGoSource）：只动 ImportSpec 路径与
//	             `const modulePath`，绝不动其它字面量/注释/原始描述符；
//	其它文本  —— 保留字面量替换（Makefile/Dockerfile/scripts/* 的路径与产物名）。
func rewriteFileContent(base, content, module, name string) (string, bool, error) {
	switch {
	case base == "go.mod":
		out := applyRewriteRules(content, module, name, func(scope string) bool { return scope != "go" })
		out = rewriteModuleDirective(out, module)
		return out, out != content, nil
	case strings.HasSuffix(base, ".go"):
		return rewriteGoSource(content, module)
	default:
		out := applyRewriteRules(content, module, name, func(scope string) bool { return scope == "all" })
		return out, out != content, nil
	}
}

// rewriteGoSource 对 .go 源码做 import 感知的受控重写，返回（新源码, 是否有改动, 错误）。
//
// 为什么不能沿用字面量替换：`.pb.go` 的 protobuf raw descriptor 是**带长度前缀**的字符串
// 字面量，例如 `"\x1cproto/jimu/v1/userinfo.proto\x12\x07..."`（0x1c = 28 = 该路径长度）。
// 字面量替换会把 `proto/jimu/v1/...` 改成 `proto/<module>/v1/...` 而长度字节仍是 28，
// 于是 protobuf 反序列化越界 —— 进程一启动就 panic（filedesc.unmarshalSeed:
// slice bounds out of range）。只重写 ImportSpec 路径即从根上消除这类误改。
//
// 实现用 AST 定位 + **原文字节替换**（不做 printer 往返）：除被改写的字面量外逐字节不变。
func rewriteGoSource(src, module string) (string, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, baseName("rewrite"), strings.NewReader(src), parser.ParseComments)
	if err != nil {
		return "", false, fmt.Errorf("parse go source: %w", err)
	}
	offsetOf := func(p token.Pos) int { return fset.PositionFor(p, false).Offset }
	var edits []textEdit
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		rest, ok := strings.CutPrefix(path, "jimu/")
		if !ok {
			continue
		}
		quoted := strconv.Quote(module + "/" + rest)
		if quoted == spec.Path.Value {
			continue // to == from：不动点，不能登记为改动（幂等契约）
		}
		edits = append(edits, textEdit{
			start: offsetOf(spec.Path.Pos()), end: offsetOf(spec.Path.End()),
			text: quoted,
		})
	}
	// tools/{composereport,checkcapabilities} 的 `const modulePath = "jimu"`（无斜杠，import 规则
	// 抓不到）；按**声明名 + 值**精确定位，不碰其它同值字面量。
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, id := range vs.Names {
				if id.Name != "modulePath" || i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if v, err := strconv.Unquote(lit.Value); err != nil || v != "jimu" {
					continue
				}
				quoted := strconv.Quote(module)
				if quoted == lit.Value {
					continue
				}
				edits = append(edits, textEdit{
					start: offsetOf(lit.Pos()), end: offsetOf(lit.End()),
					text: quoted,
				})
			}
		}
	}
	if len(edits) == 0 {
		return src, false, nil
	}
	formatted, err := format.Source([]byte(applyTextEdits(src, edits)))
	if err != nil {
		return "", false, fmt.Errorf("format rewritten go source: %w", err)
	}
	return string(formatted), true, nil
}

// applyRewriteRules 按 scope 判定应用规则表（前一条的输出是后一条的输入）；scopeApplies 为 nil
// 表示不做作用域过滤（不动点校验用，取保守的超集）。
func applyRewriteRules(content, module, name string, scopeApplies func(scope string) bool) string {
	for _, rule := range rewriteRules {
		if scopeApplies != nil && !scopeApplies(rule.scope) {
			continue
		}
		content = strings.ReplaceAll(content, rule.literal, rule.repl(module, name))
	}
	return content
}

// modulePathCollides 报告 to 自身是否仍会被规则表命中。命中意味着字面量替换在该模块路径上不幂等：
// 第二遍会把已写入的前缀再改写一次（to="github.com/foo/jimu"），甚至单遍即坏
// （to="github.com/foo/jimu/v2" 被 "jimu/" 规则二次插入前缀）。to 等于源前缀本身时放行。
func modulePathCollides(to, name string) bool {
	probe := to + "/"
	return applyRewriteRules(probe, to, name, nil) != probe
}

// ModulePathCollides exposes the fail-closed module path predicate to the
// generator facade and its legacy framework helpers.
func ModulePathCollides(to, name string) bool { return modulePathCollides(to, name) }

// rewriteModuleDirective 把 go.mod 的 module 指令指向 to，保留行尾形态与行尾注释
// （规则表的 "module jimu\n" 字面量只覆盖「LF + 尾换行 + 无注释」这一常见形态）。
func rewriteModuleDirective(content, module string) string {
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		body, eol := line, ""
		if strings.HasSuffix(body, "\n") {
			body, eol = strings.TrimSuffix(body, "\n"), "\n"
			if strings.HasSuffix(body, "\r") {
				body, eol = strings.TrimSuffix(body, "\r"), "\r\n"
			}
		}
		m := goModuleDirective.FindStringSubmatch(body)
		if m == nil {
			continue
		}
		repl := "module " + module
		if m[1] != "" {
			repl += " " + m[1]
		}
		lines[i] = repl + eol
	}
	return strings.Join(lines, "")
}

// moduleDirectiveValue 取 go.mod 里 module 指令的值（module 后的第一个 token）；找不到返回 ""。
// 与 goModuleDirective 相互独立：即使兜底正则没匹配上，也能发现「仍是 module jimu」并报错。
func moduleDirectiveValue(content string) string {
	m := goModuleValue.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return m[1]
}

// ruleApplies 判定规则作用域：gomod 只认 go.mod，go 只认 .go，all 认全部白名单文本文件。
func ruleApplies(scope, base string) bool {
	switch scope {
	case "gomod":
		return base == "go.mod"
	case "go":
		return strings.HasSuffix(base, ".go")
	default:
		return true
	}
}

// isRewriteTarget 判定文件是否属于可重写文本（后缀白名单或 Makefile/Dockerfile）。
func isRewriteTarget(base string) bool {
	if base == "Makefile" || base == "Dockerfile" {
		return true
	}
	return textExtensions[strings.ToLower(filepath.Ext(base))]
}

// IsRewriteTarget reports whether a file participates in module rewriting.
func IsRewriteTarget(base string) bool { return isRewriteTarget(base) }

// artifactName 把模块路径规范化为产物名：小写，非 [a-z0-9-] 折叠为 "-"，去首尾 "-"。
func artifactName(module string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(path.Base(module)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	return strings.Trim(b.String(), "-")
}

// ArtifactName exposes the generated binary name derivation used by the
// facade's validation path.
func ArtifactName(module string) string { return artifactName(module) }

func relPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}

// RelPath returns a stable slash-separated path relative to root.
func RelPath(root, p string) string { return relPath(root, p) }
