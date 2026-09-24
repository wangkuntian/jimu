package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/capabilities/catalog"
)

// 单形态渲染（S8 的确定性重渲染）：生成 internal/profiles/registry/registry.go、
// internal/profiles/<shape>/assembly.go、internal/profiles/<shape>/drivers.go、
// internal/profiles/active/assembly.go。四份文件的内容都由「能力集 + 驱动集」纯函数决定，
// 因此 `jimu capability add` 可以整体重渲染而不是手改 AST。

// shapeCapability 是单形态清单里的一项（模板输入）。
type shapeCapability struct {
	Alias   string // 显式包别名：<name>module（usermodule/queuemodule/…），与标准库零冲突
	Path    string // 框架仓里的 import 路径（随 module 重写一起改写）
	Ungated bool
	Drivers []string
}

// HasDrivers 是否有选中的驱动（模板里控制 `Drivers: []string{...}` 字段）。
func (c shapeCapability) HasDrivers() bool { return len(c.Drivers) > 0 }

// configFiles 返回 configs 渲染产物的相对路径。
func configFiles() []string {
	return []string{"configs/app.yaml", "configs/app.prod.yaml"}
}

// catalogFiles 返回 catalog 渲染产物的相对路径。
func catalogFiles() []string {
	return []string{
		"internal/capabilities/catalog/catalog.go",
		"internal/capabilities/catalog/migration.go",
	}
}

// shapeFiles 返回四份单形态产物的相对路径（按渲染顺序）。
func shapeFiles(set CapabilitySet) []string {
	return []string{
		"internal/profiles/registry/registry.go",
		filepath.ToSlash(filepath.Join("internal/profiles", set.Shape, "assembly.go")),
		filepath.ToSlash(filepath.Join("internal/profiles", set.Shape, "drivers.go")),
		"internal/profiles/active/assembly.go",
	}
}

// RenderShape 渲染四份单形态产物到 dst。能力顺序沿用 set.Declared（catalog 拓扑序 = 装配顺序的
// 合法序列，端口提供者先于消费者）。迁移携带能力（MigrationOnly）不参与装配，故不进 assembly。
func RenderShape(root, dst string, set CapabilitySet) error {
	_ = root
	caps := make([]shapeCapability, 0, len(set.Declared))
	for _, name := range set.Declared {
		caps = append(caps, shapeCapability{
			Alias:   name + "module",
			Path:    frameworkModule + "/" + capabilityDirPrefix + "/" + name,
			Ungated: slices.Contains(set.Ungated, name),
			Drivers: set.Drivers[name],
		})
	}
	data := struct {
		Shape        string
		Capabilities []shapeCapability
	}{Shape: set.Shape, Capabilities: caps}

	// 用**有序切片**而不是 map：map 字面量里 keys 相同会静默丢弃后一个产物（`--shape=active`
	// 时 assembly.go 与 active/assembly.go 同名，曾被无声吞掉）。现在显式检测冲突并报错。
	files := []struct{ rel, tpl string }{
		{"internal/profiles/registry/registry.go", "project/registry.go.tmpl"},
		{filepath.ToSlash(filepath.Join("internal/profiles", set.Shape, "assembly.go")), "project/assembly.go.tmpl"},
		{filepath.ToSlash(filepath.Join("internal/profiles", set.Shape, "drivers.go")), "project/drivers.go.tmpl"},
		{"internal/profiles/active/assembly.go", "project/active.go.tmpl"},
	}
	seen := make(map[string]string, len(files))
	for _, f := range files {
		if prev, ok := seen[f.rel]; ok {
			return fmt.Errorf("render shape %q: 产物路径 %s 冲突（模板 %s 与 %s 撞车）—— --shape 不得取保留名", set.Shape, f.rel, prev, f.tpl)
		}
		seen[f.rel] = f.tpl
		text, err := Template(f.tpl)
		if err != nil {
			return err
		}
		out, err := RenderText(f.rel, text, data)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(f.rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", f.rel, err)
		}
		if err := os.WriteFile(target, out, goFileMode); err != nil {
			return fmt.Errorf("write %s: %w", f.rel, err)
		}
	}
	return nil
}

// renderShape 是 generateInto 里的落点包装。
func renderShape(root, dst string, set CapabilitySet) error {
	return RenderShape(root, dst, set)
}

// RenderCatalog 渲染生成项目的 internal/capabilities/catalog/{catalog.go,migration.go}：
// entries = 选定集 ∪ **迁移携带能力**（Minor 7 裁定：S2 原话「迁移携带能力不进 entries」作废
// —— tenant 的 Descriptor 必须进 entries，否则 filterAll/MigrationSet 取不到，
// `jimu migrate` 会漏 tenant 的建表/加列，正是 P2.6 C1 的生成项目版。T3 不得按旧口径改回
// migrationExtras）；known = 框架全量能力名 catalog 18 ∪ Ungated 7（S5）。
//
// NOTE: T3 将替换本渲染器 —— entries/MigrationSchemaDeps 的完整口径（Configs/Assets 等）
// 由 T3 收口；当前版本只是「可编译、可跑 check-capabilities」的骨架，后续任务请**替换**而非叠加。
func RenderCatalog(dst string, set CapabilitySet) error {
	type entry struct {
		Alias string
		Path  string
	}
	type dep struct {
		From string
		To   string
	}
	data := struct {
		Capabilities []entry
		Known        []string
		Deps         []dep
	}{}
	for _, name := range set.Declared {
		data.Capabilities = append(data.Capabilities, entry{
			Alias: name + "module",
			Path:  frameworkModule + "/" + capabilityDirPrefix + "/" + name,
		})
	}
	// 迁移携带能力也要被 catalog import（它只提供 Descriptor/Migrations，不参与装配）。
	for _, name := range set.MigrationOnly {
		data.Capabilities = append(data.Capabilities, entry{
			Alias: name + "module",
			Path:  frameworkModule + "/" + capabilityDirPrefix + "/" + name,
		})
	}
	// Minor 6/S5：known = 框架全量能力名（catalog 18 ∪ Ungated 7），语义是「软依赖指向缺席能力
	// = 降级」，不是笔误。
	data.Known = set.Known
	for _, name := range set.Declared {
		deps := catalog.MigrationSchemaDeps[name]
		if len(deps) == 0 {
			continue
		}
		quoted := make([]string, 0, len(deps))
		for _, d := range deps {
			quoted = append(quoted, fmt.Sprintf("%q", d))
		}
		data.Deps = append(data.Deps, dep{From: name, To: "[]string{" + strings.Join(quoted, ", ") + "}"})
	}
	files := map[string]string{
		"internal/capabilities/catalog/catalog.go":   "project/catalog.go.tmpl",
		"internal/capabilities/catalog/migration.go": "project/catalog_migration.go.tmpl",
	}
	for rel, tpl := range files {
		text, err := Template(tpl)
		if err != nil {
			return err
		}
		out, err := RenderText(rel, text, data)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(target, out, goFileMode); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}

// RenderConfigs 渲染生成项目的 configs/*.yaml。
//
// NOTE: T4 将替换本渲染器（当前为原样复制 configs/*.yaml 的占位）—— T4 的段选择器只保留
// 内核段 + 选中能力的段，且 app.yaml 与 app.prod.yaml 同口径（S6）。请**替换**而非叠加。
func RenderConfigs(root, dst string) error {
	for _, rel := range configFiles() {
		if err := copyOneFile(root, dst, rel); err != nil {
			return err
		}
	}
	return nil
}

// docsRelDir 是 apidocs 的编译期依赖目录（apidocs/swagger.go 里 `_ "jimu/docs/openapi"`）。
const docsRelDir = "docs/openapi"

// RenderDocs 渲染生成项目的 docs/openapi（apidocs 的**编译期**依赖：apidocs/swagger.go 里
// `_ "jimu/docs/openapi"` 是硬 import，缺了 apidocs 一选就编译不过）。
//
// NOTE: T6 将替换本渲染器 —— 资产（deploy/** 与 docs/openapi）的完整口径归 T6；这里是
// 「apidocs 选中则携带 docs/openapi」的最小落地，请**替换**而非叠加。
func RenderDocs(root, dst string, set CapabilitySet) error {
	if !slices.Contains(set.Copy, "apidocs") {
		return nil
	}
	if _, _, err := CopyTree(filepath.Join(root, filepath.FromSlash(docsRelDir)),
		filepath.Join(dst, filepath.FromSlash(docsRelDir)), nil); err != nil {
		return fmt.Errorf("copy docs/openapi: %w", err)
	}
	return nil
}
