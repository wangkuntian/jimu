package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
