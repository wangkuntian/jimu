package generator

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/capability"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/profiles/registry"
)

// CapabilitySet 是一次生成的全部选择结果，是「复制什么」的唯一输入。
type CapabilitySet struct {
	Shape         string              // 形态名：--profile 的名字，或 --shape（默认 "app"）
	Profile       string              // 非空表示来自 --profile（驱动集取自形态清单）
	Declared      []string            // 选定集（含 Ungated 非 catalog 条目；catalog 拓扑序 + Ungated 追加）
	Known         []string            // 软依赖错别字检查的全量能力名（catalog 18 ∪ Ungated 7，Minor 6/S5）
	Copy          []string            // 实际复制的能力根包目录（S1：声明集 ∪ 编译闭包 ∪ schema 依赖闭包）
	Roots         []string            // 出货二进制的「能力根包」import 闭包（golden 口径，见 CapabilityRoots）
	MigrationOnly []string            // 只为 schema 依赖而复制的能力（S2：只带 migrations + domain + 生成的 module.go）
	DomainOnly    []string            // 只为核心编译期依赖而复制的能力（裁定④：只带 domain/）
	Drivers       map[string][]string // 能力名 → 选中驱动（S2/S4）
	Ungated       []string            // 声明集里的非 catalog（Ungated）条目，渲染 assembly 时置 Ungated: true
}

// defaultShape 是 --with 不带 --shape 时的形态名：生成项目仍需要一个形态目录名。
const defaultShape = "app"

// shapeIdentifier 是 --shape 的合法形式：小写字母开头的小写标识符（对应 internal/profiles/<name> 目录名）。
var shapeIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// reservedShapes 是与生成目录结构/固定产物冲突的名字：除 `internal/profiles/<shape>/` 外，
// registry 与 active 是 profiles 下的两个固定子目录（`--shape=registry` 会生成自 import 的
// registry 包、`--shape=active` 会让 assembly.go 路径与 active 产物相撞），其余是生成树里的结构名。
var reservedShapes = map[string]bool{
	"registry": true,
	"active":   true,
	"catalog":  true,
	"configs":  true,
	"profiles": true,
	"internal": true,
	"cmd":      true,
}

// validateShape 校验 --shape：保留名或非标识符一律 fail-closed（否则会生成自 import 的包、
// 或让两份产物写到同一路径后静默丢文件）。
func validateShape(shape string) error {
	if shape == "" {
		return fmt.Errorf("--shape 不得为空")
	}
	if !shapeIdentifier.MatchString(shape) {
		return fmt.Errorf("invalid --shape %q: 只允许小写字母、数字与下划线且以字母开头（示例：app、minimal、machine）", shape)
	}
	if reservedShapes[shape] {
		return fmt.Errorf("invalid --shape %q: 是生成目录结构的保留名（registry/active/catalog/configs/profiles/internal/cmd 不可用），请换一个（示例：app、minimal、machine）", shape)
	}
	return nil
}

// frameworkModule 是框架仓自身的模块路径（S7 的发现判据 + 受控重写的源前缀）。
const frameworkModule = "jimu"

// kernelDirs 是内核必需目录（S1 的原样复制集）：不含任何能力目录，也不含 profiles 的形态目录
// （registry/<shape>/active 由 RenderShape 生成，profiles.go 单列在 kernelFiles）。
var kernelDirs = []string{
	"internal/kernel",
	"internal/app",
	"internal/assembly",
	"internal/capability",
	"internal/contract",
	"internal/config",
	"internal/shared",
	"internal/e2e",
	"conf",
	"cmd/server",
	"cmd/cli",
}

// kernelFiles 是内核必需的单文件（相对框架仓根）。
var kernelFiles = []string{
	"internal/profiles/profiles.go",
	"go.mod",
	"go.sum",
	".golangci.yml",
	".gitignore",
}

// ParseCapabilitySet 解析 --profile / --with（互斥）+ --shape。
//
//	--profile=<name>：能力集 = registry.Lookup(name) 的清单（含 Ungated）；
//	                 驱动集 = 该清单 assembly.Capability.Drivers。
//	--with=a,b[:drv]：能力集 = capability.Resolve(框架全量能力, names)（Requires 闭包 + 拓扑序；
//	                 catalog 18 ∪ Ungated 7 都可选，Ungated 渲染时置 Ungated: true）；
//	                 驱动集 = 各能力 Descriptor.Drivers 首项（S4），冒号后缀逐项覆盖。
//
// 两者都给 → 报错 mutually exclusive；都不给 → 报错（要求二选一）。两者都合法：
// 参数校验是唯一来源，--dry-run 也走同一条路径。
// Copy/MigrationOnly 由 CapabilityRoots 补齐（需 root，见 CapabilityRoots）。
func ParseCapabilitySet(profile, with, shape string) (CapabilitySet, error) {
	if profile != "" && with != "" {
		return CapabilitySet{}, fmt.Errorf("--profile and --with are mutually exclusive")
	}
	if profile == "" && with == "" {
		return CapabilitySet{}, fmt.Errorf("one of --profile or --with is required")
	}
	if shape != "" {
		// 即使 --profile 会覆盖形态名，也先校验用户显式给的 --shape：fail-closed 报错优于静默忽略。
		if err := validateShape(shape); err != nil {
			return CapabilitySet{}, err
		}
	}
	set := CapabilitySet{Shape: shape, Drivers: map[string][]string{}}
	if set.Shape == "" {
		set.Shape = defaultShape
	}
	root, err := frameworkRoot()
	if err != nil {
		return CapabilitySet{}, err
	}
	// --with 的能力名解析用**框架全量集合**（catalog 18 ∪ Ungated 7，Important 3）：
	// Ungated 条目仍不受 capabilities.enabled 门控，只是可以被单独选中。
	descs, inCatalog, err := capabilityDescriptors(root)
	if err != nil {
		return CapabilitySet{}, err
	}
	byName := make(map[string]contract.Descriptor, len(descs))
	for _, d := range descs {
		byName[d.Name] = d
	}
	if profile != "" {
		a, err := registry.Lookup(profile)
		if err != nil {
			return CapabilitySet{}, err
		}
		set.Profile = profile
		set.Shape = profile
		declared := make(map[string]bool, len(a.Capabilities))
		for _, c := range a.Capabilities {
			name := c.Descriptor.Name
			declared[name] = true
			if len(c.Drivers) > 0 {
				set.Drivers[name] = slices.Clone(c.Drivers)
			}
		}
		// 声明集按 catalog 拓扑序（registry 的清单是装配顺序，不能直接当 entries）：
		// 先 catalog 条目，再补非 catalog（Ungated）条目 —— Ungated 没有迁移/权限点，
		// 只需在 assembly 里置 Ungated: true，位置由拓扑序的相对关系决定（它们无依赖）。
		for _, d := range catalog.All() {
			if declared[d.Name] {
				set.Declared = append(set.Declared, d.Name)
			}
		}
		for _, c := range a.Capabilities {
			name := c.Descriptor.Name
			if !c.Ungated || inCatalog[name] {
				continue
			}
			if _, ok := byName[name]; !ok {
				return CapabilitySet{}, fmt.Errorf("profile %q declares unknown capability %q", profile, name)
			}
			set.Declared = append(set.Declared, name)
			set.Ungated = append(set.Ungated, name)
		}
	} else {
		if err := validateShape(set.Shape); err != nil {
			return CapabilitySet{}, err
		}
		names, overrides, err := parseWith(with)
		if err != nil {
			return CapabilitySet{}, err
		}
		desc, err := capability.Resolve(descs, names)
		if err != nil {
			return CapabilitySet{}, err
		}
		for _, d := range desc {
			set.Declared = append(set.Declared, d.Name)
			if !inCatalog[d.Name] {
				set.Ungated = append(set.Ungated, d.Name)
			}
			// S4：--with 没有形态清单可依，选中驱动默认取 Descriptor.Drivers 首项
			// （catalog 与 Ungated 同规则：storage→local、dataops→csv）。
			if len(d.Drivers) > 0 {
				set.Drivers[d.Name] = []string{d.Drivers[0]}
			}
		}
		for name, drv := range overrides {
			d := byName[name]
			if !slices.Contains(d.Drivers, drv) {
				return CapabilitySet{}, fmt.Errorf("unknown driver %q for capability %q (available: %s)",
					drv, name, strings.Join(d.Drivers, ", "))
			}
			set.Drivers[name] = []string{drv}
		}
	}
	return CapabilityRoots(root, set)
}

// parseWith 解析 `--with` 的 `<cap>[:<drv>][,<cap>[:<drv>]]…` 语法，返回能力名清单与
// 显式驱动覆盖（能力 → 驱动）。空项与空驱动名都是 fail-closed 的语法错误。
func parseWith(with string) ([]string, map[string]string, error) {
	names := make([]string, 0, 4)
	overrides := map[string]string{}
	for _, item := range strings.Split(with, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, nil, fmt.Errorf("--with contains an empty capability")
		}
		name, drv, hasDrv := strings.Cut(item, ":")
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, nil, fmt.Errorf("--with contains an empty capability")
		}
		names = append(names, name)
		if !hasDrv {
			continue
		}
		drv = strings.TrimSpace(drv)
		if drv == "" {
			return nil, nil, fmt.Errorf("capability %q has an empty driver", name)
		}
		if prev, ok := overrides[name]; ok && prev != drv {
			return nil, nil, fmt.Errorf("capability %q selects two drivers (%q and %q)", name, prev, drv)
		}
		overrides[name] = drv
	}
	return names, overrides, nil
}

// CapabilityRoots 在框架仓 root 上求「能力根包闭包」并把 schema 依赖落成 MigrationOnly
// （S1/S2）、把内核编译期 domain 依赖落成 DomainOnly（裁定 ④）。复制集口径：
//
//	Copy = 声明集 ∪ 编译闭包(S1) ∪ schema 依赖(S2) ∪ 内核编译期 domain 依赖(④)。S2 修正（控制者裁定）：迁移携带目录 = migrations/ + domain/ + 生成的 module.go；
//
// **domain 必须带**，因为内核 internal/app 的 seed 代码硬 import
// capabilities/<cap>/domain（叶子包，`app → 能力 domain` 是仓内既已记录的偏差），
// 只带 migrations 会让生成项目编译失败。application/infrastructure/interfaces/wire/cli
// 仍不复制（不需要，且会把整条能力链拖进来）。口径与 tools/checkcapabilities 的 closureImports 相同（packages.Load +
// Tests=false + 只保留 jimu/internal/capabilities/<cap> 这一层）；闭包只从 Declared 出发，
// 迁移携带目录（MigrationOnly）是 S2 生成的无 import stub，展开它们会把整条能力链拖进来。
func CapabilityRoots(root string, set CapabilitySet) (CapabilitySet, error) {
	descs, _, err := capabilityDescriptors(root)
	if err != nil {
		return CapabilitySet{}, err
	}
	requires := make(map[string][]string, len(descs))
	known := make(map[string]bool, len(descs))
	for _, d := range descs {
		requires[d.Name] = d.Requires
		known[d.Name] = true
	}
	set.Known = knownCapabilityNames(descs)
	closure, roots, err := capabilityClosure(root, set.Declared, requires, known)
	if err != nil {
		return CapabilitySet{}, err
	}
	set.Roots = roots
	declared := map[string]bool{}
	for _, name := range set.Declared {
		if declared[name] {
			continue
		}
		declared[name] = true
		dir := filepath.Join(root, capabilityDirPrefix, name)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return CapabilitySet{}, fmt.Errorf("capability %q has no source directory %s", name, filepath.ToSlash(dir))
		}
	}
	only := map[string]bool{}
	for name := range declared {
		for _, dep := range catalog.MigrationSchemaDeps[name] {
			if !declared[dep] {
				only[dep] = true
			}
		}
	}
	// Copy 是**并集**：编译闭包 ∪ 迁移携带目录 ∪ 内核编译期 domain 依赖（裁定 ④），
	// 故去重后排序。Copy 只用于「复制哪些能力目录」，内容口径由 MigrationOnly/DomainOnly 决定。
	copySet := map[string]bool{}
	covered := map[string]bool{} // 已经带 domain/ 的能力（完整复制或迁移携带）
	for _, name := range closure {
		copySet[name] = true
		covered[name] = true
	}
	for name := range only {
		copySet[name] = true
		covered[name] = true
	}
	domainOnly := map[string]bool{}
	for _, entry := range kernelRequiredDomains {
		name, _, _ := strings.Cut(entry, "/")
		if covered[name] {
			continue // 该能力的 domain/ 已经在完整复制或迁移携带里
		}
		domainOnly[name] = true
		copySet[name] = true
	}
	set.MigrationOnly = slices.Sorted(maps.Keys(only))
	set.DomainOnly = slices.Sorted(maps.Keys(domainOnly))
	set.Copy = slices.Sorted(maps.Keys(copySet))
	return set, nil
}

const capabilityDirPrefix = "internal/capabilities"

// kernelRequiredDomains 是**内核编译期 domain 依赖**（裁定 ④）：`internal/app`（`seed.go`）
// 在编译期直接 import 的能力 domain 叶子包 —— 实测来源（`go list -deps` 交叉核对）：
//
//	internal/app/seed.go → internal/capabilities/{access,tenant,user}/domain
//
// 「app → 能力 domain 叶子包」是仓内既已记录并接受的偏差，与 S1 的编译闭包**同性质**：
// 无论是否选中对应能力，这三个 `domain/` 子树都必须复制，否则生成项目编译失败
// （`--with=queue` 曾经因此坏掉）。只复制 `domain/`（模型），不复制这三个能力的其它子包。
//
// 防漂移：`TestKernelRequiredDomainsCoverAppImports` 解析 `internal/app/*.go` 的 import 与本表
// 比对 —— 将来有人往 app 里再加一个能力 domain 而没同步本表，该测试会红。
var kernelRequiredDomains = []string{"access/domain", "tenant/domain", "user/domain"}

// capabilityDescriptors 返回**框架全量能力描述符**：catalog 的 18 项 + 7 个 Ungated
// （apidocs/storage/notification/retention/ws/grpc/encryption）从源码读出的声明。
// `--with` 的能力名解析用这一份全量集合（Important 3）：Ungated 仍不受 `capabilities.enabled`
// 门控（P2.4 裁定），只是可以被 `jimu new` 单独选中。
func capabilityDescriptors(root string) ([]contract.Descriptor, map[string]bool, error) {
	all := catalog.All()
	inCatalog := make(map[string]bool, len(all))
	for _, d := range all {
		inCatalog[d.Name] = true
	}
	extra, err := nonCatalogDescriptors(root, inCatalog)
	if err != nil {
		return nil, nil, err
	}
	return append(all, extra...), inCatalog, nil
}

// nonCatalogDescriptors 扫描 internal/capabilities/*，把**不在 catalog 里**但导出
// `var Descriptor` 的能力读成描述符（只取 Name/Requires/Drivers —— 选择与驱动过滤所需的三项）。
func nonCatalogDescriptors(root string, inCatalog map[string]bool) ([]contract.Descriptor, error) {
	dir := filepath.Join(root, filepath.FromSlash(capabilityDirPrefix))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath.ToSlash(dir), err)
	}
	var out []contract.Descriptor
	for _, e := range entries {
		if !e.IsDir() || inCatalog[e.Name()] || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		capDir := filepath.Join(dir, e.Name())
		file, err := descriptorFile(capDir)
		if err != nil {
			return nil, fmt.Errorf("capability %s: %w", e.Name(), err)
		}
		if file == "" {
			continue // 不是能力包（如 catalog 自身、辅助包）
		}
		requires, err := descriptorSliceField(file, "Requires")
		if err != nil {
			return nil, err
		}
		drivers, err := descriptorSliceField(file, "Drivers")
		if err != nil {
			return nil, err
		}
		out = append(out, contract.Descriptor{Name: e.Name(), Requires: requires, Drivers: drivers})
	}
	slices.SortFunc(out, func(a, b contract.Descriptor) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// knownCapabilityNames 返回软依赖错别字检查的全量能力名（Minor 6/S5）：catalog 18 ∪ Ungated 7。
func knownCapabilityNames(descs []contract.Descriptor) []string {
	out := make([]string, 0, len(descs))
	for _, d := range descs {
		out = append(out, d.Name)
	}
	return out
}

// capabilityClosure 求 declared 的「能力根包」import 闭包：packages.Load 载入生产包图
// （Tests=false），把三种 import 都映射回**属主能力根包**（Fix round 3 / C1）：
//
//	`capabilities/<cap>`            → <cap>（根包 import）
//	`capabilities/<cap>/<非 domain>` → <cap>（子包 import 也是编译期依赖；如 auth → user/infrastructure）
//	`capabilities/<cap>/domain...`   → **忽略**（domain 由裁定④/S2 按「只带 domain/」处理，
//	                                   否则 tenant/auth 链会被整条拖进来）
//
// 之后再对结果传递地补 `Requires`：完整复制的成员必须带上它的硬依赖（auth → user、access），
// 否则只剩 root 包而缺子包/依赖，`go build` 失败（--with=breach 就是这样坏的）。
//
// 返回两个集合（T8 拆分，裁定 17）：
//
//	closure：**复制集**口径（属主映射 + Requires 传递补齐）—— 决定哪些能力目录要落盘；
//	roots：  **出货二进制**口径 —— 只统计 import 路径恰为 `capabilities/<cap>` 的**根包**
//	         （不含子包，与 scripts/check_profiles.sh 的 cap_roots 逐值同口径），**不做**
//	         Requires 传递补齐。它才是 check_profiles.sh golden 的真值：硬依赖只保证目录被复制、
//	         装配可用，**不保证被 import** —— 如 `--with=mfa` 的 `auth → access`，access 在复制集里
//	         但出货二进制里没有它的根包（T8 的独立 oracle 发现：golden 曾多写一个 access，
//	         生成项目自己的 `make profiles-check` 会红）。
func capabilityClosure(root string, declared []string, requires map[string][]string, known map[string]bool) (closure, roots []string, err error) {
	if len(declared) == 0 {
		return nil, nil, nil
	}
	dir := filepath.Join(absPath(root), filepath.FromSlash(capabilityDirPrefix))
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:   dir,
		Tests: false,
	}
	// 逐个能力根包加载，**不能**用 "./..."：目录模式会把哪些包纳入闭包取决于当前目录树，
	// 且 pattern 的初始包与 Tests=false 的交互在不同目录形态下并不一致（实测 ./... 会把
	// 测试变体见到、甚至把兄弟能力包当成初始包纳入）。显式 pattern 让口径确定：闭包恒等于
	// 「这些能力根包的生产 import 闭包」。
	patterns := make([]string, 0, len(declared))
	for _, name := range declared {
		patterns = append(patterns, "./"+name)
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, nil, fmt.Errorf("load capability import graph under %s: %w", filepath.ToSlash(dir), err)
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("no packages matched any capability root package under %s", filepath.ToSlash(dir))
	}
	// 判据用 **import 路径**前缀（jimu/internal/capabilities/<cap>），不是文件系统路径：
	// packages.Package.PkgPath 是 import 路径。
	prefix := frameworkModule + "/" + capabilityDirPrefix + "/"
	found := map[string]bool{}
	rootSet := map[string]bool{}
	var loadErrs []string
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrs = append(loadErrs, e.Error())
		}
		rest, ok := strings.CutPrefix(p.PkgPath, prefix)
		if !ok {
			return
		}
		name, sub, hasSub := strings.Cut(rest, "/")
		if hasSub && (sub == "domain" || strings.HasPrefix(sub, "domain/")) {
			// domain 叶子包不是「该能力参与装配」的证据：它由裁定④/S2 决定只带 domain/。
			return
		}
		if !known[name] {
			return
		}
		found[name] = true
		if !hasSub {
			rootSet[name] = true // 根包 = 出货二进制口径（cap_roots）
		}
	})
	if len(loadErrs) > 0 {
		return nil, nil, fmt.Errorf("load capability import graph: %s", strings.Join(loadErrs, "; "))
	}
	// 传递补齐硬依赖：完整复制的成员必须带上它的 Requires（auth → user、access）。C1。
	for changed := true; changed; {
		changed = false
		for name := range found {
			for _, dep := range requires[name] {
				if !found[dep] {
					found[dep] = true
					changed = true
				}
			}
		}
	}
	out := make([]string, 0, len(found))
	for name := range found {
		out = append(out, name)
	}
	slices.Sort(out)
	return out, slices.Sorted(maps.Keys(rootSet)), nil
}

// FrameworkRoot 是 frameworkRoot 的导出形式：供测试与后续任务（`jimu capability add`、
// 模板门禁）复用同一套「cwd 向上找 module jimu」的源根发现口径（S7）。
func FrameworkRoot() string {
	root, err := frameworkRoot()
	if err != nil {
		return ""
	}
	return root
}

// frameworkRoot 从 cwd 向上找第一个 go.mod 且其 module 行为 jimu 的目录（S7）。
//
// 搜索层数与 `internal/config` 找 `configs/` 的口径**完全一致**（`config.SearchDepthUp`，含 cwd
// 本身）：两处深度不一致会出现半路失败 —— 源根找到了、`--report` 的 ProbeAssembly 却因为向上
// 5 层内没有 `configs/` 而加载不到能力配置段。找不到即 fail-closed 报错（绝不静默拿空源目录
// 生成空项目），错误里带上层数与起点便于定位。
func frameworkRoot() (string, error) {
	var root string
	err := config.WithWorkingDirectory("", func() error {
		start, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
		dir := start
		for i := 0; i < config.SearchDepthUp; i++ {
			if mod, err := moduleOf(filepath.Join(dir, "go.mod")); err == nil && mod == frameworkModule {
				root = dir
				return nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return fmt.Errorf("no framework source root within %d levels up from %s: run jimu from the framework checkout (or at most %d levels below its root), whose go.mod declares module %s",
			config.SearchDepthUp, filepathSlash(start), config.SearchDepthUp-1, frameworkModule)
	})
	return root, err
}

var goModuleLine = regexp.MustCompile(`(?m)^[ \t]*module[ \t]+([^ \t\r\n]+)`)

// moduleOf 取 go.mod 的 module 指令值；文件不存在/无法解析时返回错误。
func moduleOf(gomod string) (string, error) {
	content, err := os.ReadFile(gomod)
	if err != nil {
		return "", err
	}
	m := goModuleLine.FindSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("%s has no module directive", filepath.ToSlash(gomod))
	}
	return string(m[1]), nil
}
