package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/profiles/full"
	"jimu/tools/internal/profileassets"
	"jimu/tools/internal/projectmetrics"
)

// 本文件实现 `jimu new --report`（第 2 节裁定 8）：把**生成本身**的编译面写进
// <dir>/docs/profiles/generated-report.md。
//
// 口径与 docs/profiles/compose-report.md **同一份实现**（tools/internal/projectmetrics），
// 但含义不同：本仓报告是「5 个形态横向对比」（含「相对 full」归一化列、构建期 overlay 叙述），
// 生成项目在生成期就把形态固定下来，只有一个形态、没有叠加，因此报告里没有归一化列、没有
// overlay，主模块名就是项目自己。

// reportRelPath 报告落点（相对生成项目根）。
const reportRelPath = "docs/profiles/generated-report.md"

// Report 度量生成项目 root 的编译面：文件数（闭包）/代码行（闭包）/迁移数/表数/路由数/
// 重型依赖/解析集。
//
// 能力集从 <root>/.jimu-generated 读回（marker 存的是**声明集**，T3 裁定 7）：装配集与
// `jimu capability add` 的重渲染同源，故 `--report` 可以在任意时刻对已生成项目重跑。
// go.mod 直接依赖与资产/生成文件数由 WriteReport 补齐（它们是「这个目录」的度量，不是装配面）。
func Report(root string) (*projectmetrics.Metrics, error) {
	m, err := LoadMarker(root)
	if err != nil {
		return nil, err
	}
	src, err := frameworkRoot()
	if err != nil {
		return nil, err
	}
	return reportFor(root, src, m)
}

// reportFor 用**调用方已解析的框架源根**度量 `root`：`capability add` 的 cwd 就是生成项目
// （文档里的用法 `cd proj && jimu capability add x`），此时 cwd 里没有 `module jimu`，只能把
// `--from`/`marker.SourceRoot` 解析出的源根传进来 —— 不能重新按 cwd 发现（Fix round 2 修掉的
// 真实 bug：刷新报告时静默失败并打告警）。
func reportFor(root, src string, m *Marker) (*projectmetrics.Metrics, error) {
	set, err := markerSet(src, m)
	if err != nil {
		return nil, err
	}
	asm, err := assemblyFor(set)
	if err != nil {
		return nil, err
	}
	// overlay 传 nil：生成项目是单形态提交态（internal/profiles/active 已指向唯一形态），
	// 没有构建期叠加 —— 这正是生成版 compose-report 与报告口径的差别所在。
	metrics, err := projectmetrics.Of(root, m.Module, asm, nil)
	if err != nil {
		return nil, fmt.Errorf("measure %s: %w（--report 需要框架仓的 configs/：ProbeAssembly 会加载能力配置段，查找口径是 cwd 向上 %d 层内（含 cwd）的 configs/，与源根发现同一常量；请从框架仓根目录或其 %d 层以内运行）", filepathSlash(root), err, config.SearchDepthUp, config.SearchDepthUp-1)
	}
	return &metrics, nil
}

// markerSet 用 marker 的声明集/驱动集重建 CapabilitySet（不新增能力、不做依赖补全）。
//
// 填 Declared（装配集与解析集）、Shape/Profile（报告表头）、Drivers（marker 原样）、Ungated
// （按 catalog 成员关系重算），再经 **CapabilityRoots** 重算 Copy/MigrationOnly/DomainOnly/
// Known —— 与 `jimu capability add` 的重渲染同一口径（T3 裁定 7：marker 只存声明集）。这样
// 「对已生成项目独立重跑报告」得到的「迁移携带能力 / 只带 domain 的能力」两行与生成时逐值一致
// （不重建的话它们会退化成 `-`），报告因此在重跑下仍逐字节幂等。
func markerSet(root string, m *Marker) (CapabilitySet, error) {
	descs, inCatalog, err := capabilityDescriptors(root)
	if err != nil {
		return CapabilitySet{}, err
	}
	known := make(map[string]bool, len(descs))
	for _, d := range descs {
		known[d.Name] = true
	}
	set := CapabilitySet{
		Shape:    m.Shape,
		Profile:  m.Profile,
		Declared: slices.Clone(m.Capabilities),
		Drivers:  cloneDrivers(m.Drivers),
	}
	for _, name := range set.Declared {
		if !known[name] {
			return CapabilitySet{}, fmt.Errorf("%s records unknown capability %q", markerFile, name)
		}
		if !inCatalog[name] {
			set.Ungated = append(set.Ungated, name)
		}
	}
	return CapabilityRoots(root, set)
}

// assemblyFor 返回 set 的**进程内**装配清单：从框架 `full` 形态的清单里按 set.Declared 取出对应项。
//
// 为什么不在生成器里直接 import 各能力包：那会把 25 个能力的 Wire 拖进 CLI 的编译面，而生成器
// 只需要「某个选择会注册哪些路由/迁移」这一个事实。`full` 是全量装配，且 cmd/cli 已经经
// internal/profiles/registry 链接了它，因此按名取子集是零新增依赖的唯一取法。
//
// 驱动字段沿用 full 的声明（路由数不受驱动选择影响）；装配顺序 = set.Declared（catalog 拓扑序 +
// Ungated 追加），与生成项目渲染出的 assembly.go 同序。
func assemblyFor(set CapabilitySet) (assembly.Assembly, error) {
	fullAsm := full.Assembly()
	byName := make(map[string]assembly.Capability, len(fullAsm.Capabilities))
	for _, c := range fullAsm.Capabilities {
		byName[c.Descriptor.Name] = c
	}
	caps := make([]assembly.Capability, 0, len(set.Declared))
	for _, name := range set.Declared {
		c, ok := byName[name]
		if !ok {
			return assembly.Assembly{}, fmt.Errorf("capability %q has no entry in the framework full assembly", name)
		}
		caps = append(caps, c)
	}
	return assembly.Assembly{Name: set.Shape, Capabilities: caps}, nil
}

// WriteReport 把报告写到 <dir>/docs/profiles/generated-report.md，并把同一份内容打印到 stdout。
//
// `set` 提供报告表头需要的声明集/迁移携带集/驱动集（它们不是 Metrics 的字段）；module 与资产根
// 从 marker 读回 —— 报告的每个数字都必须来自**这个目录**，不靠调用方转述。
func WriteReport(dir string, m projectmetrics.Metrics, set CapabilitySet) error {
	content, err := writeReportFileContent(dir, m, set)
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(content)
	return err
}

// writeReportFile 与 WriteReport 同源但**不打印**：`capability add` 的静默刷新用它（add 的输出
// 已经有自己的「改了哪些文件」清单，再打一整份报告是噪音）。
func writeReportFile(dir string, m projectmetrics.Metrics, set CapabilitySet) error {
	_, err := writeReportFileContent(dir, m, set)
	return err
}

// writeReportFileContent 渲染并落盘，返回渲染文本（打印与静默写共用同一份内容）。
func writeReportFileContent(dir string, m projectmetrics.Metrics, set CapabilitySet) (string, error) {
	marker, err := LoadMarker(dir)
	if err != nil {
		return "", err
	}
	deps, err := projectmetrics.DirectDeps(dir, marker.Module)
	if err != nil {
		return "", fmt.Errorf("count direct dependencies: %w", err)
	}
	generated, assetFiles, err := generatedTreeCounts(dir)
	if err != nil {
		return "", err
	}
	content := renderGeneratedReport(m, set, marker, deps, generated, assetFiles)
	target := filepath.Join(dir, filepath.FromSlash(reportRelPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("create directory for %s: %w", reportRelPath, err)
	}
	if err := writeFile(target, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", reportRelPath, err)
	}
	return content, nil
}

// generatedTreeCounts 返回（生成文件数，资产文件数）：前者是生成树里的全部文件（不含
// `.jimu-generated` 标记与报告自身），后者是复制进来的资产根（profileassets.AssetRoots：
// `deploy`、`docs/openapi`）下的文件数。
func generatedTreeCounts(dir string) (generated, assetFiles int, err error) {
	roots := profileassets.AssetRoots()
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		rel := relPath(dir, p)
		if rel == markerFile || rel == reportRelPath {
			return nil
		}
		generated++
		for _, root := range roots {
			if rel == root || strings.HasPrefix(rel, root+"/") {
				assetFiles++
				break
			}
		}
		return nil
	})
	return generated, assetFiles, err
}

// renderGeneratedReport 渲染生成项目报告。输出只由实测值与 marker 决定（无时间戳、无绝对路径），
// 因此重跑同一项目产出逐字节相同的文件。
func renderGeneratedReport(m projectmetrics.Metrics, set CapabilitySet, marker *Marker, deps, generated, assetFiles int) string {
	var b strings.Builder
	b.WriteString("# 生成项目编译面报告\n\n")
	b.WriteString("> 由 `jimu new --report`（`tools/generator`）生成，**请勿手工编辑**：能力集变化后重新生成并提交本文件。\n")
	b.WriteString("> 指标口径与 `docs/profiles/compose-report.md` 同源（共享 `tools/internal/projectmetrics`），\n")
	b.WriteString("> 但**含义不同**：那份报告是 5 个形态的横向对比；本项目在生成期就把形态固定下来 ——\n")
	b.WriteString("> 只有一个形态、没有叠加，因此下表没有任何归一化列，每个数字都是本项目自己的实测值，\n")
	b.WriteString("> 不与任何东西比较。\n\n")

	b.WriteString("## 项目概况\n\n")
	b.WriteString("| 项 | 值 |\n|---|---|\n")
	fmt.Fprintf(&b, "| 模块 | `%s` |\n", marker.Module)
	fmt.Fprintf(&b, "| 形态 | `%s` |\n", set.Shape)
	fmt.Fprintf(&b, "| 声明能力（装配集） | %s |\n", joinCell(set.Declared))
	fmt.Fprintf(&b, "| 迁移携带能力（schema 依赖） | %s |\n", joinCell(set.MigrationOnly))
	fmt.Fprintf(&b, "| 只带 domain 的能力（内核编译期依赖） | %s |\n", joinCell(set.DomainOnly))
	fmt.Fprintf(&b, "| 选中驱动 | %s |\n", joinCell(flattenDrivers(set.Drivers)))
	fmt.Fprintf(&b, "| 复制的资产根 | %s |\n", joinCell(marker.Assets))
	b.WriteString("\n")

	b.WriteString("## 指标口径\n\n")
	b.WriteString("与 `docs/profiles/compose-report.md` 的「指标口径」段同一份实现（`tools/internal/projectmetrics`）：\n\n")
	b.WriteString("| 指标 | 口径 |\n|---|---|\n")
	b.WriteString("| 生成文件数 | 生成树里的全部文件数（不含 `.jimu-generated` 标记与本报告自身） |\n")
	b.WriteString("| 本模块 Go 文件 / 代码行（闭包） | `golang.org/x/tools/go/packages` 载入 `./cmd/server` 的 import 闭包，只统计本模块的非 `_test.go` 文件（生成项目的形态在生成期固定，直接读提交态选点） |\n")
	b.WriteString("| go.mod 直接依赖 | `go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all` 的非空行数（不含主模块自身；`--report` 在 `go mod tidy` **之后**度量） |\n")
	b.WriteString("| 迁移数 | **迁移集**（声明集 ∪ schema 依赖，与 `jimu migrate` 同一口径）各 `Descriptor.Migrations` 中 `migrations/mysql/*.sql` 的文件数（postgres 同名同数） |\n")
	b.WriteString("| 表数 | **迁移集**各 `Descriptor.Owns` 的并集大小（口径同迁移数） |\n")
	b.WriteString("| 路由数 | 装配集在裸 `gin.Engine` 上 `RegisterHTTP` 后的 `r.Routes()` 条数（不启动监听、不连库） |\n")
	b.WriteString("| 重型依赖 | 闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |\n")
	b.WriteString("| 资产文件数 | 复制进来的资产根（`deploy/`、`docs/openapi/`）下的文件数 |\n\n")

	b.WriteString("## 编译面\n\n")
	b.WriteString("| 指标 | 值 |\n|---|---:|\n")
	fmt.Fprintf(&b, "| 生成文件数 | %d |\n", generated)
	fmt.Fprintf(&b, "| 本模块 Go 文件（闭包） | %d |\n", m.Files)
	fmt.Fprintf(&b, "| 本模块代码行（闭包） | %d |\n", m.Lines)
	fmt.Fprintf(&b, "| go.mod 直接依赖 | %d |\n", deps)
	fmt.Fprintf(&b, "| 迁移数 | %d |\n", m.Migrations)
	fmt.Fprintf(&b, "| 表数 | %d |\n", m.Tables)
	fmt.Fprintf(&b, "| 路由数 | %d |\n", m.Routes)
	fmt.Fprintf(&b, "| 重型依赖 | %s |\n", joinCell(m.HeavyDeps))
	fmt.Fprintf(&b, "| 资产文件数 | %d |\n", assetFiles)
	b.WriteString("\n")

	b.WriteString("## 解析集（`jimu migrate` / `seed` 的能力清单）\n\n")
	fmt.Fprintf(&b, "| 装配的能力（按装配顺序） |\n|---|\n| %s |\n", joinCell(m.Capabilities))
	return b.String()
}

// joinCell 渲染表格单元：空集为 `-`，否则逗号分隔。
func joinCell(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}
