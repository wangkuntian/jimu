package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"jimu/internal/capabilities/catalog"
)

// 生成项目的 catalog（T3）：把「能力集」渲染成 internal/capabilities/catalog/{catalog.go,
// migration.go}。模板蓝本是**本仓真实文件** internal/capabilities/catalog/{catalog.go,migration.go}
// —— 导出符号与函数体逐字同源（All/Names/ValidateDeclarations/Degraded/Resolve 的薄封装、
// MigrationSchemaDeps/WithMigrationSchemaDeps/MigrationSet/filterAll），只有三处差异：
//
//	① entries 收窄为「选定集 ∪ 迁移携带能力」；
//	② knownNames 固化为**框架全量**能力名（catalog 全量 ∪ 非 catalog 条目，S5）；
//	③ import 别名一律 <name>module。
//
// entries 必须含迁移携带能力（如 tenant）：filterAll/MigrationSet 只从 entries 取 descriptor，
// 漏掉它们会让 `jimu migrate` 少建表/少加列（P2.6 的 C1 在生成项目里复活）—— 这是 T2 审查后的
// 裁定修订第 1 条（S2 的「迁移携带能力不进 entries」作废），**不得**改回 migrationExtras 形态。

// CapEntry 是生成版 catalog 的一条 import + entries 项。
type CapEntry struct {
	Name  string // 能力名（用于查 MigrationSchemaDeps）
	Alias string // 显式包别名：<name>module（usermodule/queuemodule/…），与标准库零冲突
	Path  string // 框架仓 import 路径（随 module 重写一起改写）
}

// SchemaDep 是 MigrationSchemaDeps 的一行；To 已渲染成 Go 字面量（如 `{"tenant"}`）。
type SchemaDep struct {
	From string
	To   string
}

// CatalogData 是 catalog 模板的输入。SchemaDeps 用有序切片而不是 map：map 遍历顺序不确定，
// 会让生成产物在不同运行间抖动（黄金对比与 `jimu capability add` 的重渲染都会受影响）。
type CatalogData struct {
	Entries       []CapEntry  // entries = 选定集 ∪ 迁移携带能力，catalog 拓扑序 + Ungated 追加
	MigrationOnly []CapEntry  // 只随迁移复制、不参与装配的能力（entries 的子集，供注释点明）
	KnownNames    []string    // 框架全量能力名（catalog 全量 ∪ 非 catalog 条目）
	SchemaDeps    []SchemaDep // 只保留「键 ∈ entries」的行
}

// catalogOutputs 是 catalog 渲染产物的「相对路径 → 模板」有序清单。
var catalogOutputs = []struct{ rel, tpl string }{
	{"internal/capabilities/catalog/catalog.go", "project/catalog.go.tmpl"},
	{"internal/capabilities/catalog/migration.go", "project/catalog_migration.go.tmpl"},
}

// catalogFiles 返回 catalog 渲染产物的相对路径（--dry-run 计数与落盘同源，不会漂移）。
func catalogFiles() []string {
	out := make([]string, 0, len(catalogOutputs))
	for _, o := range catalogOutputs {
		out = append(out, o.rel)
	}
	return out
}

// RenderCatalog 渲染生成项目的 internal/capabilities/catalog/{catalog.go,migration.go}，返回
// 「相对路径 → 文件内容」。渲染成内存产物而不是直接落盘：黄金文件对比（T3）与后续
// `jimu capability add` 的确定性重渲染都需要纯函数形态。
func RenderCatalog(set CapabilitySet) (map[string][]byte, error) {
	data, err := catalogData(set)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(catalogOutputs))
	for _, o := range catalogOutputs {
		text, err := Template(o.tpl)
		if err != nil {
			return nil, err
		}
		rendered, err := RenderText(o.rel, text, data)
		if err != nil {
			return nil, err
		}
		out[o.rel] = rendered
	}
	return out, nil
}

// renderCatalog 是 renderDerivedAll 里的落点包装（与 renderShape 同形）：RenderCatalog 的产物写进
// 生成目录。接入点收敛在此一处，new.go 不再直接拼装/写盘。
func renderCatalog(dst string, set CapabilitySet) error {
	files, err := RenderCatalog(set)
	if err != nil {
		return err
	}
	for _, o := range catalogOutputs {
		content, ok := files[o.rel]
		if !ok {
			return fmt.Errorf("render catalog: 缺少产物 %s", o.rel)
		}
		target := filepath.Join(dst, filepath.FromSlash(o.rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", o.rel, err)
		}
		if err := os.WriteFile(target, content, goFileMode); err != nil {
			return fmt.Errorf("write %s: %w", o.rel, err)
		}
	}
	return nil
}

// catalogCoversAll 报告本次选择是否覆盖框架**全量** catalog。
//
// entries = 「框架 catalog ∩ (选定集 ∪ 迁移携带)」（见 catalogData），恒为子集，所以「覆盖全量」
// 等价于「框架每个 catalog 能力都在 selected 里」—— `--profile=full` 这类项目成立，`--with=queue`
// 这类不成立。
//
// 它只服务一件事：测试树裁剪里的「组成依赖」（见 pruneUnsatisfiableTests）。读 `catalog.All()` 的
// 测试把**框架全量组成**当期望值，项目 catalog 是子集时会运行期失败（`--with=queue` 时
// `internal/app/seed_test.go` 的 seed 不再查权限，sqlmock 期望落空）；而选择恰好覆盖全量时那些
// 期望值成立，按同一规则裁掉就是白白丢覆盖（full 类项目会少 3 个测试文件）。于是按此条件化。
func catalogCoversAll(set CapabilitySet) bool {
	selected := make(map[string]bool, len(set.Declared)+len(set.MigrationOnly))
	for _, name := range set.Declared {
		selected[name] = true
	}
	for _, name := range set.MigrationOnly {
		selected[name] = true
	}
	for _, d := range catalog.All() {
		if !selected[d.Name] {
			return false
		}
	}
	return true
}

// catalogData 把能力集折算成模板输入，保证与框架仓 catalog 同构的拓扑序。
//
// fail-closed（Fix round 1 / Minor 1）：knownNames 是 ValidateDeclarations 的错别字白名单，空集
// 会让**所有**软依赖都被判 unknown（生成项目一启动就炸）；entries 里出现白名单之外的能力名同理
// —— 两者都直接报错，而不是渲染出一份「能编译但语义错」的清单。这里只做自洽断言，不写死总数
// （框架加第 26 个能力时不应误伤）。
func catalogData(set CapabilitySet) (CatalogData, error) {
	if len(set.Known) == 0 {
		return CatalogData{}, fmt.Errorf("render catalog: knownNames 为空 —— CapabilitySet 必须经 ParseCapabilitySet 构造（known = 框架全量能力名）")
	}
	all := catalog.All()
	inCatalog := make(map[string]bool, len(all))
	for _, d := range all {
		inCatalog[d.Name] = true
	}
	selected := make(map[string]bool, len(set.Declared)+len(set.MigrationOnly))
	for _, name := range set.Declared {
		selected[name] = true
	}
	for _, name := range set.MigrationOnly {
		selected[name] = true
	}

	data := CatalogData{KnownNames: slices.Clone(set.Known)}
	// entries：先 catalog 拓扑序（选定集 ∪ 迁移携带能力），再按声明顺序追加非 catalog（Ungated）
	// 条目 —— Ungated 无依赖无迁移，追加在尾部不破坏拓扑序（与 layout.go 的 Declared 同规则）。
	for _, d := range all {
		if selected[d.Name] {
			data.Entries = append(data.Entries, capEntry(d.Name))
		}
	}
	for _, name := range set.Declared {
		if !inCatalog[name] {
			data.Entries = append(data.Entries, capEntry(name))
		}
	}
	// entries 必须落在 knownNames 白名单内：否则生成项目的 ValidateDeclarations 会把这份清单
	// 自己的条目判成「未知能力」（子集清单本身就是从全量集合里选出来的，出现白名单外的名字
	// 只可能是 CapabilitySet 构造错了）。
	known := make(map[string]bool, len(set.Known))
	for _, name := range set.Known {
		known[name] = true
	}
	for _, e := range data.Entries {
		if !known[e.Name] {
			return CatalogData{}, fmt.Errorf("render catalog: entries 含 knownNames 白名单外的能力 %q —— knownNames 必须是框架全量能力名", e.Name)
		}
	}
	for _, d := range all {
		if slices.Contains(set.MigrationOnly, d.Name) {
			data.MigrationOnly = append(data.MigrationOnly, capEntry(d.Name))
		}
	}
	// SchemaDeps 按 entries 顺序输出 —— 「键 ∈ entries」的过滤在此天然成立。
	for _, e := range data.Entries {
		deps := catalog.MigrationSchemaDeps[e.Name]
		if len(deps) == 0 {
			continue
		}
		quoted := make([]string, 0, len(deps))
		for _, dep := range deps {
			quoted = append(quoted, strconv.Quote(dep))
		}
		data.SchemaDeps = append(data.SchemaDeps, SchemaDep{From: e.Name, To: "{" + strings.Join(quoted, ", ") + "}"})
	}
	return data, nil
}

// capEntry 由能力名构造模板条目（别名规则与 RenderShape 的 shapeCapability 一致）。
func capEntry(name string) CapEntry {
	return CapEntry{
		Name:  name,
		Alias: name + "module",
		Path:  frameworkModule + "/" + capabilityDirPrefix + "/" + name,
	}
}
