package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// descriptorLiteral 在 AST 里定位 `var Descriptor = contract.Descriptor{...}`，返回其复合字面量。
// 找不到即报错（fail-closed：宁可不生成，也不静默漏收窄 Drivers）。
func descriptorLiteral(fset *token.FileSet, file *ast.File) (*ast.CompositeLit, error) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "Descriptor" || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("var Descriptor is not a composite literal")
			}
			return lit, nil
		}
	}
	return nil, fmt.Errorf("no var Descriptor composite literal found")
}

// descriptorInfo 是一份能力根包源码里的 Descriptor 视图：既能读字段字面量（S2 生成
// module.go 时搬 Owns），也能就地改写（收窄 Drivers）。
type descriptorInfo struct {
	fset  *token.FileSet
	file  *ast.File
	src   string
	lit   *ast.CompositeLit
	descr *ast.CompositeLit
}

// loadDescriptor 解析一份 Go 源码并定位 Descriptor 字面量。
func loadDescriptor(src string) (*descriptorInfo, error) {
	fset := token.NewFileSet()
	// 用 io.Reader 传源码：go/parser 的 src 参数语义陷阱 —— string/[]byte 形式**必须以换行
	// 开头**才被当作源码，否则整串被当成文件名（实测报 "file name too long"）；而加换行前缀
	// 又会让 token 偏移整体 +1，本类型用 PositionFor 反解字节区间取字面量，会取错一位。
	file, err := parser.ParseFile(fset, baseName("module"), strings.NewReader(src), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	lit, err := descriptorLiteral(fset, file)
	if err != nil {
		return nil, err
	}
	// contract.Descriptor{...} 是外层 SelectorExpr 的复合字面量；裸 Descriptor{...} 也容忍。
	descr := lit
	if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name != "Descriptor" {
		return nil, fmt.Errorf("var Descriptor has unexpected type")
	}
	return &descriptorInfo{fset: fset, file: file, src: src, lit: lit, descr: descr}, nil
}

// field 返回 Descriptor 字面量里某个字段的值表达式（不存在返回 nil）。
func (d *descriptorInfo) field(name string) ast.Expr {
	for _, elt := range d.descr.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == name {
			return kv.Value
		}
	}
	return nil
}

// literal 返回某个字段的**源码字面量**（不存在返回 ""）：用于把框架仓的 Owns 原样搬进
// S2 生成的 module.go，不手写。
func (d *descriptorInfo) literal(name string) string {
	value := d.field(name)
	if value == nil {
		return ""
	}
	return d.source(value)
}

// setField 替换某个字段的值为 expr 源码；字段不存在则追加（保住其余字段与注释）。
// path 取 **base 名**：printer 会在 KeyValueExpr 的 Key 前插一个空格（对齐用），
// 用 path.Base 消除它（与 format.go 的 fieldList 同口径）。
func (d *descriptorInfo) setField(name, expr string) error {
	value, err := parser.ParseExprFrom(d.fset, baseName(name), expr, 0)
	if err != nil {
		return fmt.Errorf("parse %s value %q: %w", name, expr, err)
	}
	if existing := d.field(name); existing != nil {
		for i, elt := range d.descr.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok && kv.Value == existing {
				d.descr.Elts[i] = &ast.KeyValueExpr{Key: kv.Key, Value: value}
				return nil
			}
		}
	}
	d.descr.Elts = append(d.descr.Elts, &ast.KeyValueExpr{Key: ast.NewIdent(name), Value: value})
	return nil
}

// baseName 给 ParseExprFrom 一个不会与文件里其它位置冲突的伪文件名。
func baseName(name string) string { return "jimu_descriptor_" + name }

// removeField 删除某个字段（连同该行）。
func (d *descriptorInfo) removeField(name string) {
	for i, elt := range d.descr.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == name {
			d.descr.Elts = append(d.descr.Elts[:i], d.descr.Elts[i+1:]...)
			return
		}
	}
}

// setSlice 把某个切片字段替换为给定元素（空集则删除字段）。
func (d *descriptorInfo) setSlice(name string, items []string) error {
	if len(items) == 0 {
		d.removeField(name)
		return nil
	}
	return d.setField(name, "[]string{"+quoteList(items)+"}")
}

// source 返回某个节点在原文里的字面量。
func (d *descriptorInfo) source(node ast.Node) string {
	return d.src[d.fset.PositionFor(node.Pos(), false).Offset:d.fset.PositionFor(node.End(), false).Offset]
}

// bytes 返回改写后的 Go 源码。打印时**丢弃位置信息**（printer.Config.Mode 不含 UsePositions）：
// 文件里没有制表符、但预格式化代码保留了原有的列对齐空白，若按位置打印会得到
// `Name:         "access",` 这类带多余空格的文本；RawFormat 下 printer 会按自身规则重新对齐。
func (d *descriptorInfo) bytes() ([]byte, error) {
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.RawFormat, Tabwidth: 8}
	if err := cfg.Fprint(&buf, d.fset, d.file); err != nil {
		return nil, fmt.Errorf("print patched descriptor: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format patched descriptor: %w", err)
	}
	return formatted, nil
}

// goFileMode 是生成 .go 文件的统一权限位。
const goFileMode fs.FileMode = 0o644

// descriptorFieldFromFile 读框架仓某能力根包文件，取 Descriptor 里指定 string 字段的源码字面量。
func descriptorFieldFromFile(file, field string) (string, bool, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", false, err
	}
	d, err := loadDescriptor(string(content))
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", filepathSlash(file), err)
	}
	value := d.literal(field)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

// patchDrivers 把能力根包文件里的 `Descriptor.Drivers` 收窄为选中驱动集：空集删除该字段。
// 不收窄就必红 —— 生成项目的 check-capabilities 断言①要求声明的每个驱动目录都存在，
// 而驱动目录按选中集过滤（S4）。
func patchDrivers(file string, drivers []string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	d, err := loadDescriptor(string(content))
	if err != nil {
		return fmt.Errorf("patch drivers in %s: %w", filepathSlash(file), err)
	}
	if err := d.setSlice("Drivers", drivers); err != nil {
		return fmt.Errorf("patch drivers in %s: %w", filepathSlash(file), err)
	}
	out, err := d.bytes()
	if err != nil {
		return fmt.Errorf("patch drivers in %s: %w", filepathSlash(file), err)
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	return os.WriteFile(file, out, info.Mode().Perm())
}

// quoteList 把字符串切片渲染成 Go 字面量元素列表（`"a", "b"`）。
func quoteList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, ", ")
}

// renderModuleOnly 构造 S2「迁移携带」能力的 module.go：只含 //go:embed migrations 与
// Descriptor{Name, Migrations, Owns}（Owns 从框架仓同名 Descriptor 复制，不手写）。
// 先拼源码文本再 format.Source：Owns 是来自框架源码的**表达式**（[]string{...}），文本模板的
// 占位符会与行尾注释里的 `%` 冲突；而 //go:embed 指令必须在变量声明的紧邻上一行，只有
// format.Source 能稳定保住这个位置（AST 打印会把注释挂到 import 声明上）。
func renderModuleOnly(descriptorSrc, name string) ([]byte, error) {
	content, err := os.ReadFile(descriptorSrc)
	if err != nil {
		return nil, err
	}
	srcInfo, err := loadDescriptor(string(content))
	if err != nil {
		return nil, err
	}
	owns := srcInfo.field("Owns")
	if owns == nil {
		return nil, fmt.Errorf("%s has no Descriptor.Owns", filepathSlash(descriptorSrc))
	}
	text := fmt.Sprintf(`%[1]spackage %[2]s

import (
	"embed"

	"jimu/internal/contract"
)

//go:embed migrations
var migrationsFS embed.FS

var Descriptor = contract.Descriptor{
	Name:       %[3]s,
	Migrations: migrationsFS,
	Owns:       %[4]s,
}
`, generatedHeader, name, strconv.Quote(name), srcInfo.source(owns))
	out, err := format.Source([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("format generated module.go for %s: %w", name, err)
	}
	return out, nil
}

// generatedHeader 是全部生成文件的统一头（S8：内容由能力集纯函数决定，手改会被重渲染覆盖）。
const generatedHeader = "// Code generated by jimu new; DO NOT EDIT.\n"

// descriptorSliceField 读框架仓某能力根包文件，取 Descriptor 里指定 []string 字段的值
// （如 Requires/Drivers）。用于把**非 catalog（Ungated）**能力的声明搬进选择集合 ——
// 这些能力不在 catalog.entries 里，只能从源码读。
func descriptorSliceField(file, field string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	d, err := loadDescriptor(string(content))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepathSlash(file), err)
	}
	literal := d.literal(field)
	if literal == "" {
		return nil, nil
	}
	expr, err := parser.ParseExpr(literal)
	if err != nil {
		return nil, fmt.Errorf("parse Descriptor.%s of %s: %w", field, filepathSlash(file), err)
	}
	var out []string
	ast.Inspect(expr, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if v, uerr := strconv.Unquote(lit.Value); uerr == nil {
			out = append(out, v)
		}
		return true
	})
	return out, nil
}
