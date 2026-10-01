package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/workspace"
	"jimu/tools/internal/profileassets"
)

// NewOptions 是 `jimu new` 的全部输入（CLI 与测试共用同一结构，参数校验只在
// ParseCapabilitySet 一处）。
type NewOptions struct {
	Dir     string // 目标目录
	Profile string // --profile：形态名（与 With 互斥）
	With    string // --with：<cap>[:<drv>][,...]（与 Profile 互斥）
	Shape   string // --shape：--with 时的形态名，默认 app
	Module  string // --module：模块路径，默认由 Dir 推导

	NoTidy bool // 跳过 go mod tidy（默认跑；--no-tidy 关闭）
	DryRun bool // 只打印计划，不落盘
	Force  bool // 只覆盖带 .jimu/manifest.json 的既有生成项目
	Report bool // 额外写 docs/profiles/generated-report.md

	// NoSelfCheck 跳过 ⑨ 自检（go build ./... + go run ./tools/checkcapabilities）。
	//
	// **只给本仓的构建类测试用**：它们在专用 GOCACHE 下自行 build/vet（见 projectbuild_test.go
	// 的 Fix round 4 —— 25 次链接写爆过共享缓存），不能让 NewProject 的自检再走默认 GOCACHE。
	// CLI 不暴露该开关：`jimu new` 恒自检。
	NoSelfCheck bool
}

// Result 是一次生成的摘要（--dry-run 与 --report 共用）。
type Result struct {
	Dir    string
	Module string
	Shape  string
	// Capabilities 是**装配集**（Declared：--profile 的形态清单，或 --with 的 Requires 闭包 + 拓扑序，
	// 与写进 marker 的 `capabilities` 同源）；CopySet 是**复制集**（Copy：装配集 ∪ 编译闭包
	// ∪ 迁移携带目录 ∪ 内核编译期 domain 依赖）。两者不等：`minimal` 装配 5 个能力，复制 8 个目录。
	// `--dry-run` 两个都打印并显式标注，避免把复制集当成装配集读。
	Capabilities []string
	CopySet      []string
	Drivers      []string
	Assets       []string
	Files        []string
	FileCount    int
	Lines        int

	// Changed 是 `capability add` 的产物差异（相对调用前的生成项目）：新增/改动/移除的生成产物
	// 相对路径（排序）。为空即「本次没有改动」（幂等）。`new` 不用该字段（它写的是全新目录）。
	Changed []string
}

// NewProject 生成项目。执行顺序与回滚语义见计划第 2 节裁定 7/8：
//
//	① 解析能力集 → ② 复制内核必需目录 → ③ 按能力复制（含驱动过滤）
//	→ ④⑤⑥ 渲染/复制全部产物 → ⑦ module 受控重写 + gofmt（在 renderDerivedAll 内完成）
//	→ ⑧ go mod tidy（--no-tidy 关闭）→ ⑨ 自检：go build ./... + go run ./tools/checkcapabilities
//
// 产物先写 <dir>.tmp-<rand>，⑦⑧⑨ **全在暂存目录里**跑，全部通过后才原子 rename；任何一步失败
// 都删除临时目录（defer RemoveAll），绝不留半成品（验收⑤）。--force 时先把既有产物 rename 成
// <dir>.old-<rand>，rename 成功后再删，避免「先删后建」留下空目录窗口。
func NewProject(opts NewOptions) (*Result, error) {
	root, err := frameworkRoot()
	if err != nil {
		return nil, err
	}
	module, err := resolveModule(opts.Dir, opts.Module)
	if err != nil {
		return nil, err
	}
	if err := validateModule(module); err != nil {
		return nil, err
	}
	doc, err := frameworkmanifest.Export(frameworkmanifest.Request{
		Root:    root,
		Profile: opts.Profile,
		With:    opts.With,
		Shape:   opts.Shape,
		Module:  module,
	})
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(doc); err != nil {
		return nil, err
	}
	if _, err := plan.Build(doc); err != nil {
		return nil, err
	}
	created, err := workspace.Create(doc, workspace.CreateOptions{
		Target:      opts.Dir,
		SourceRoot:  root,
		Module:      module,
		Force:       opts.Force,
		NoTidy:      opts.NoTidy,
		DryRun:      opts.DryRun,
		Report:      opts.Report,
		NoSelfCheck: opts.NoSelfCheck,
	})
	if err != nil {
		return nil, err
	}
	return &Result{
		Dir:          created.Target,
		Module:       created.Module,
		Shape:        doc.Selection.Shape,
		Capabilities: slices.Clone(doc.Selection.Capabilities),
		CopySet:      manifestCapabilityNames(doc),
		Drivers:      flattenDrivers(doc.Selection.Drivers),
		Assets:       manifestAssetNames(doc),
		Files:        slices.Clone(created.Files),
		FileCount:    created.FileCount,
	}, nil
}

func manifestCapabilityNames(doc manifest.Document) []string {
	values := make([]string, 0, len(doc.Capabilities))
	for _, capability := range doc.Capabilities {
		values = append(values, capability.Name)
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func manifestAssetNames(doc manifest.Document) []string {
	values := make([]string, 0, len(doc.Assets))
	for _, asset := range doc.Assets {
		values = append(values, asset.Destination)
	}
	slices.Sort(values)
	return slices.Compact(values)
}

// renderDerivedAll 把「能力集 → 全部产物」的**唯一**一条渲染/复制管线落进 dst（S8 的确定性
// 重渲染）：内核复制 → 能力复制（含驱动过滤）→ 迁移携带/内核 domain → 形态与 catalog 渲染 →
// configs 渲染 → CLI 裁剪 → tools 复制与构建文件 → module 受控重写 → 资产复制与 values.yaml
// 键裁剪 → 测试裁剪 → gofmt → 写 .jimu/manifest.json。
//
// `jimu new` 与 `jimu capability add` **共用**本函数（不得出现第二套渲染逻辑）：new 把它写到
// 空目录再整体换上；add 写到暂存目录后按差异逐文件落盘。产物只由「能力集 + module」决定，
// 因此 add 天然幂等（同一集合重跑逐字节等价）。
func renderDerivedAll(root, dst string, set CapabilitySet, module string) error {
	if err := copyKernel(root, dst); err != nil {
		return err
	}
	if err := copyCapabilities(root, dst, set); err != nil {
		return err
	}
	if err := copyMigrationOnly(root, dst, set); err != nil {
		return err
	}
	if err := copyKernelRequiredDomains(root, dst, set); err != nil {
		return err
	}
	if err := renderShape(root, dst, set); err != nil {
		return err
	}
	if err := renderCatalog(dst, set); err != nil {
		return err
	}
	if err := RenderConfigs(root, dst, set); err != nil {
		return err
	}
	if err := RenderCLIMain(root, dst, set); err != nil {
		return err
	}
	// T5：复制并定点改造 tools/**（不含 tools/generator），渲染三份单形态构建文件。
	// 必须在 RewriteModule 之前：补丁表写的是框架口径（modulePath 常量/文案），module 前缀
	// 由随后的受控重写统一改写；构建文件模板里的 jimu-* 产物名也依赖同一条重写规则。
	if err := copyToolsAndScripts(root, dst, module); err != nil {
		return err
	}
	if err := renderBuildFiles(dst, set); err != nil {
		return err
	}
	if _, err := RewriteModule(dst, frameworkModule, module); err != nil {
		return moduleError(module, err)
	}
	// T6：资产复制（deploy/** + docs/openapi 中选中能力与内核资产组实际拥有的部分）+ values.yaml
	// 顶层键裁剪（S6②，复用 T4 的 ValuesSections/RenderValuesYAML）。
	//
	// 位置有两个硬约束：
	//   - 必须在 RewriteModule **之后**：资产里的 jimu 名字是**框架自己的名字**（第 1 节裁定 3：
	//     deploy/backup/Dockerfile 的 /opt/jimu/scripts/、k8s 的 jimu-server/namespace jimu、
	//     镜像名），而字面量重写规则里有 `jimu/` → `<module>/`，先复制会把 /opt/jimu/scripts/
	//     改成 /opt/<module>/scripts/，产出一个容器内不存在的路径。资产一律逐字节复制
	//     （唯一例外是 values.yaml 的键裁剪）。
	//   - 必须在 pruneUnsatisfiableTests **之前**：apidocs/swagger.go 的生产 import
	//     `<module>/docs/openapi` 必须能在生成树里解析到，否则会被判成「复制集缺口」而报错
	//     （docs/openapi 是 apidocs 的编译期依赖）。
	assets, err := AssetsFor(set)
	if err != nil {
		return err
	}
	if _, err := CopyAssets(root, dst, assets); err != nil {
		return err
	}
	if err := renderValuesYAML(root, dst, set); err != nil {
		return err
	}
	// Important 4：按「逐文件 import 可满足性」裁剪测试树 —— 生产文件不满足 = 复制集缺口（报错），
	// 测试文件不满足 = 丢弃并记入 marker。internal/e2e/** 走同一条规则（替代 S3 的手写裁剪表）。
	// 「组成依赖」那一类按 catalogCoversAll 条件化：选择覆盖框架全量 catalog 时（如 --profile=full）
	// 那些测试的期望值成立，保留而不是白丢（P2.8 精化）。
	discarded, err := pruneUnsatisfiableTests(dst, module, assets, catalogCoversAll(set))
	if err != nil {
		return err
	}
	if err := formatTree(dst); err != nil {
		return err
	}
	_ = discarded
	return nil
}

// copyToolsAndScripts 把生成项目要用的工具树（copyTools）复制进 dst，并应用 patches.go 的
// 定点补丁（T5）。脚本与构建文件不在这里：它们由 renderBuildFiles 渲染。
//
// 过滤两条：skipTransient（临时/编辑器产物）与框架侧 `_test.go`（见 skipToolTestFile）。
func copyToolsAndScripts(root, dst, module string) error {
	for _, rel := range copyTools {
		filter := func(r string, d fs.DirEntry) bool {
			if !skipTransient(r, d) {
				return false
			}
			return d.IsDir() || !skipToolTestFile(d.Name())
		}
		if _, _, err := CopyTree(filepath.Join(root, filepath.FromSlash(rel)), filepath.Join(dst, filepath.FromSlash(rel)), filter); err != nil {
			return fmt.Errorf("copy tools directory %s: %w", rel, err)
		}
	}
	return applyFilePatches(dst, module)
}

// renderBuildFiles 渲染生成项目的三份单形态构建文件（Makefile / Dockerfile /
// scripts/check_profiles.sh）：三者都由「能力集」纯函数决定，因此 `jimu capability add` 的
// 重渲染（S8）可以整体重建它们。
func renderBuildFiles(dst string, set CapabilitySet) error {
	files := []struct {
		rel    string
		render func(CapabilitySet) ([]byte, error)
		mode   fs.FileMode
	}{
		{"Makefile", RenderMakefile, 0o644},
		{"Dockerfile", RenderDockerfile, 0o644},
		{"scripts/check_profiles.sh", RenderCheckProfiles, 0o755},
	}
	for _, f := range files {
		out, err := f.render(set)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(f.rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", f.rel, err)
		}
		if err := os.WriteFile(target, out, f.mode); err != nil {
			return fmt.Errorf("write %s: %w", f.rel, err)
		}
	}
	return nil
}

// pruneUnsatisfiableTests 裁剪生成项目的测试树，三类依赖各算一条：
//
//	import 依赖：测试文件 import 的能力包不在复制集里（未选中能力）；
//	资产依赖：测试文件读取的资产路径本次没有复制（如未选 apidocs 时的 docs/openapi）——
//	          它能编译、vet 也过，只是运行期 os.ReadFile 失败；
//	组成依赖：测试文件读**生成期派生的组成清单** `internal/capabilities/catalog`（`catalog.All()`
//	          / `Resolve` / `MigrationSet` …）—— 生成项目的 catalog 是**本项目专属子集**，以它为
//	          期望值的测试断言的是框架全量组成，在裁剪项目里会运行期失败（实测：`--with=queue`
//	          时 `internal/app/seed_test.go` 的 4 个 TestRunSeed_* 把全量 permissions 序列当成
//	          期望，sqlmock 期望落空；`internal/kernel/db/*_migration_integration_test.go` 同理）。
//	          **条件化（P2.8 精化）**：catalogComplete 为真时（选择覆盖框架全量 catalog，如
//	          `--profile=full`）那些期望值成立，这类文件**保留**而不是白丢 —— 判定见
//	          catalogCoversAll。
//
// 口径是「**逐文件**丢弃 + 同测试包回退」（计划 Task 2 §2 原文即逐文件过滤）：
//
//	① 逐文件：每个测试文件独立判定，不可满足的**只丢它自己**（不动同目录其它可满足的文件）；
//	② 包回退：同一**测试包**（`package foo` 与 `package foo_test` 是两个包）里若有文件被丢、
//	   而某个**保留**文件引用了被丢文件的顶层符号（func/var/const/type），逐文件保留会留下
//	   `undefined: xxx` —— 此时该测试包**整组**丢弃（这是 internal/e2e 的形态：
//	   admin_routes_parity_test.go 引用被丢文件里的 newTestAppWithDB）。
//
// 生产文件不满足 → 一律报错（这是 C1「复制闭包不完整」的捕获网，宁可失败也不产出编译不过的项目）。
// 只删测试、**不扩复制集**：未选中能力/未复制资产不会因为某个测试引用它而被拉进生成项目。
//
// 回退判据用 AST 顶层声明名 ∩ 保留文件的标识符集合（保守、确定性、不依赖工具链）：同名局部变量
// 或别的包的导出同名声字面量会**多丢**（安全方向），不会**少丢**（少丢才会产出编译不过的项目）。
func pruneUnsatisfiableTests(dst, module string, assets []string, catalogComplete bool) ([]string, error) {
	pkgs := map[string]bool{}
	var testRels []string
	err := filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel := relPath(dst, p)
		if strings.HasSuffix(d.Name(), "_test.go") {
			testRels = append(testRels, rel)
			return nil
		}
		pkgs[filepath.ToSlash(filepath.Dir(rel))] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(testRels)

	prefix := module + "/"
	absent := missingAssets(assets)
	deps := make(map[string]testDeps, len(testRels)) // 相对路径 → 依赖画像
	type groupKey struct{ dir, pkg string }
	groups := map[groupKey][]string{}
	for _, rel := range testRels {
		d, aerr := analyzeTestFile(filepath.Join(dst, filepath.FromSlash(rel)), prefix, pkgs, absent)
		if aerr != nil {
			return nil, aerr
		}
		deps[rel] = d
		key := groupKey{dir: filepath.ToSlash(filepath.Dir(rel)), pkg: d.pkg}
		groups[key] = append(groups[key], rel)
	}

	drop := map[string]bool{}
	for _, key := range slices.SortedFunc(maps.Keys(groups), func(a, b groupKey) int {
		if a.dir != b.dir {
			return strings.Compare(a.dir, b.dir)
		}
		return strings.Compare(a.pkg, b.pkg)
	}) {
		files := groups[key]
		var unsatisfied []string
		droppedDecls := map[string]bool{}
		for _, rel := range files {
			if deps[rel].satisfied(catalogComplete) {
				continue
			}
			unsatisfied = append(unsatisfied, rel)
			for name := range deps[rel].decls {
				droppedDecls[name] = true
			}
		}
		if len(unsatisfied) == 0 {
			continue
		}
		cascade := false
		for _, rel := range files {
			if !deps[rel].satisfied(catalogComplete) {
				continue
			}
			for name := range deps[rel].refs {
				if droppedDecls[name] {
					cascade = true
					break
				}
			}
			if cascade {
				break
			}
		}
		if cascade {
			for _, rel := range files {
				drop[rel] = true
			}
			continue
		}
		for _, rel := range unsatisfied {
			drop[rel] = true
		}
	}

	discarded := make([]string, 0, len(drop))
	for _, rel := range slices.Sorted(maps.Keys(drop)) {
		if rerr := os.Remove(filepath.Join(dst, filepath.FromSlash(rel))); rerr != nil {
			return nil, fmt.Errorf("remove unsatisfiable test %s: %w", rel, rerr)
		}
		discarded = append(discarded, rel)
	}
	if err := assertNoProductionGap(dst, prefix, pkgs); err != nil {
		return nil, err
	}
	return discarded, nil
}

// assertNoProductionGap 扫生产文件的 import 缺口（错误要精确到文件，不能被测试裁剪吞掉）。
func assertNoProductionGap(dst, prefix string, pkgs map[string]bool) error {
	return filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		missing, merr := missingModuleImports(p, prefix, pkgs)
		if merr != nil {
			return merr
		}
		if len(missing) > 0 {
			return fmt.Errorf("复制集缺口：生产文件 %s import %s，但生成树里没有该包", relPath(dst, p), strings.Join(missing, ", "))
		}
		return nil
	})
}

// compositionManifestDir 是生成期**派生**的组成清单目录（相对模块根）：生成项目里的
// `internal/capabilities/catalog` 只含本项目选中的能力，读它的测试把「框架全量组成」当期望值，
// 属不可移植（见 pruneUnsatisfiableTests 的「组成依赖」）。
const compositionManifestDir = "internal/capabilities/catalog"

// testDeps 是一个测试文件的静态依赖画像（一次解析得出）：
//
//	missingImports：指向本模块但生成树里没有的 import（未选中能力）；
//	missingAssets： 引用的、本次未复制的资产前缀（如 docs/openapi）；
//	decls：         顶层声明名（func/var/const/type；方法不算，其接收者类型名已覆盖）；
//	refs：          文件里出现的全部标识符（同测试包回退判据用）。
type testDeps struct {
	pkg            string
	missingImports []string
	missingAssets  []string
	compositionDep bool // 读生成期派生的组成清单（internal/capabilities/catalog）
	decls          map[string]bool
	refs           map[string]bool
}

// satisfied 逐文件可满足性：不带缺失的 import、不引用未复制的资产，且（生成 catalog 只是框架全量
// 的**子集**时）不依赖组成清单的**值** —— catalogComplete 为真时后者不再构成不可满足（见
// pruneUnsatisfiableTests 的「组成依赖」条件化）。
func (d testDeps) satisfied(catalogComplete bool) bool {
	return len(d.missingImports) == 0 && len(d.missingAssets) == 0 &&
		(catalogComplete || !d.compositionDep)
}

// analyzeTestFile 解析一次测试文件得到 testDeps（src 传 nil：让 go/parser 从**文件名**读盘）。
func analyzeTestFile(file, prefix string, pkgs map[string]bool, absent []string) (testDeps, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
	if err != nil {
		return testDeps{}, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	d := testDeps{pkg: parsed.Name.Name, decls: map[string]bool{}, refs: map[string]bool{}}
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if path == prefix+compositionManifestDir {
			d.compositionDep = true
			continue
		}
		rel, ok := strings.CutPrefix(path, prefix)
		if ok && !pkgs[rel] {
			d.missingImports = append(d.missingImports, path)
		}
	}
	d.missingAssets = assetRefsOf(parsed, absent)
	for _, decl := range parsed.Decls {
		switch v := decl.(type) {
		case *ast.FuncDecl:
			if v.Recv == nil {
				d.decls[v.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range v.Specs {
				switch t := spec.(type) {
				case *ast.ValueSpec:
					for _, n := range t.Names {
						recordDecl(d, n.Name)
					}
				case *ast.TypeSpec:
					recordDecl(d, t.Name.Name)
				}
			}
		}
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name != "_" {
			d.refs[id.Name] = true
		}
		return true
	})
	return d, nil
}

// recordDecl 记录顶层声明名。空标识符 `_` 除外：`var _ T = ...` 之类的声明名是 `_`，而几乎每个
// 文件都含 `_`（`import _ "x"`、`_ = y`），把它当符号会让回退判据**处处误触发**（T8 Fix round 1
// 实测：internal/app 的 seed_test.go 因此被连坐丢弃）。
func recordDecl(d testDeps, name string) {
	if name == "_" {
		return
	}
	d.decls[name] = true
}

// missingAssets 返回「框架声明了、但本次没有复制」的资产路径（如未选 apidocs 时的
// `docs/openapi`）。摘要口径与 AssetsFor 同源（profileassets.Declared 的前缀集合）。
func missingAssets(copied []string) []string {
	have := make(map[string]bool, len(copied))
	for _, p := range copied {
		have[profileassets.Canonical(p)] = true
	}
	var missing []string
	for _, paths := range profileassets.Declared() {
		for _, p := range paths {
			canonical := profileassets.Canonical(p)
			if canonical == "" || have[canonical] {
				continue
			}
			missing = append(missing, canonical)
		}
	}
	slices.Sort(missing)
	return slices.Compact(missing)
}

// missingAssetRefs 返回 file 引用的缺席资产路径（相对仓库根的资产前缀，去重排序）。与
// analyzeTestFile 共用 assetRefsOf 一份判定；独立入口供单测直接钉「两种字面量写法都认」。
func missingAssetRefs(file string, absent []string) ([]string, error) {
	if len(absent) == 0 {
		return nil, nil
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	return assetRefsOf(parsed, absent), nil
}

// assetRefsOf 从已解析的 AST 里收集引用的缺席资产路径。两种写法都认（同一份声明在测试里的两种
// 常见读法）：
//
//	单条字面量：`"docs/openapi/swagger.json"`、`"../../docs/openapi"`；
//	逐段字面量拼接：`filepath.Join("..", "..", "docs", "openapi", "swagger.json")`
//	（框架里的 internal/contract/openapi_test.go 正是第二种形态）。
func assetRefsOf(parsed *ast.File, absent []string) []string {
	if len(absent) == 0 {
		return nil
	}
	found := map[string]bool{}
	record := func(literal string) {
		clean := cleanAssetLiteral(literal)
		if clean == "" {
			return
		}
		for _, prefix := range absent {
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") ||
				strings.HasSuffix(clean, "/"+prefix) || strings.Contains(clean, "/"+prefix+"/") {
				found[prefix] = true
				return
			}
		}
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				if s, uerr := strconv.Unquote(v.Value); uerr == nil {
					record(s)
				}
			}
		case *ast.CallExpr:
			// filepath.Join / path.Join 的逐段字面量：全部参数都是字符串字面量时拼成一条路径，
			// 非字面量参数（变量、常量）让整条判定放弃 —— 宁可不裁，也不误裁。
			parts := make([]string, 0, len(v.Args))
			for _, arg := range v.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					parts = nil
					break
				}
				s, uerr := strconv.Unquote(lit.Value)
				if uerr != nil {
					parts = nil
					break
				}
				parts = append(parts, s)
			}
			if len(parts) > 0 {
				record(strings.Join(parts, "/"))
			}
		}
		return true
	})
	return slices.Sorted(maps.Keys(found))
}

// cleanAssetLiteral 把字面量归一化成仓库相对路径：统一斜杠、去掉开头的 `./` 与 `../` 段、
// path.Clean 折叠重复斜杠与 `.`。
func cleanAssetLiteral(literal string) string {
	p := strings.TrimSpace(filepath.ToSlash(literal))
	for {
		switch {
		case strings.HasPrefix(p, "./"):
			p = p[2:]
		case strings.HasPrefix(p, "../"):
			p = p[3:]
		default:
			return path.Clean(p)
		}
	}
}

// missingModuleImports 返回 file 里指向本模块（module/ 前缀）但生成树中不存在的 import 路径。
func missingModuleImports(file, prefix string, pkgs map[string]bool) ([]string, error) {
	fset := token.NewFileSet()
	// src 传 nil：让 go/parser 从**文件名**读盘（传 string 会被当成源码，见 astutil.go 的注释）。
	parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	var missing []string
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		rel, ok := strings.CutPrefix(path, prefix)
		if !ok {
			continue // 标准库 / 第三方：由 go.sum 与 module 缓存负责
		}
		if !pkgs[rel] {
			missing = append(missing, path)
		}
	}
	return missing, nil
}

// resolveModule 解析目标模块路径：--module 优先；否则由 Dir 的 basename 推导。
func resolveModule(dir, module string) (string, error) {
	if module != "" {
		return module, nil
	}
	base := filepath.Base(absPath(dir))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("cannot derive module path from %q; pass --module", dir)
	}
	return base, nil
}

// validateModule 在写任何东西之前校验 --module：RewriteModule 的受控重写对「自身仍含源前缀」
// 的模块路径不幂等（github.com/foo/jimu、github.com/foo/jimu/v2），这里 fail-closed 转成
// 面向用户的 --module 提示。
func validateModule(module string) error {
	if module == "" {
		return fmt.Errorf("--module 不得为空")
	}
	if modulePathCollides(module, artifactName(module)) {
		return fmt.Errorf("invalid --module %q: 不得包含 %q 前缀或以 /%s 结尾（受控重写会二次改写）",
			module, frameworkModule+"/", frameworkModule)
	}
	return nil
}

// moduleError 把 RewriteModule 的 fail-closed 报错转成面向用户的 --module 提示。
func moduleError(module string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("rewrite module %q: %w (--module 不得包含 %q 前缀或以 /%s 结尾)", module, err, frameworkModule+"/", frameworkModule)
}

// preflightTarget 落实裁定 8 的幂等/安全语义：
//   - 目标不存在，或存在但为**空目录** → 放行（`mkdir proj && jimu new proj` 是常见用法；
//     swapIntoPlace 本就把既有空目录改名让位，不需要也不应该要 --force）；
//   - 目标非空且无 `--force` → 报错 not empty；
//   - 目标非空且有 `.jimu/manifest.json` → 放行（由 swapIntoPlace 整体替换）；
//   - 目标非空但**没有** manifest → 即使带 `--force` 也拒绝（--force 的语义是「覆盖生成器产物」，
//     不是「强行写任何目录」）。
func preflightTarget(target string, force bool) error {
	entries, err := os.ReadDir(target)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect target %s: %w", filepathSlash(target), err)
	}
	if len(entries) == 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(target, ".jimu", "manifest.json")); err != nil {
		return fmt.Errorf("target directory %s is not empty and has no .jimu/manifest.json; --force only overwrites generated projects", filepathSlash(target))
	}
	if !force {
		return fmt.Errorf("target directory %s is not empty; pass --force to overwrite a generator product", filepathSlash(target))
	}
	return nil
}

// swapIntoPlace 把临时目录原子换到 target。既有 target（哪怕只是空目录：rename 到已存在的
// 目录会直接报 file exists）先改名成 <dir>.old-<rand>，新产物就位成功后再删，避免
// 「先删后建」留下空目录窗口。
func swapIntoPlace(target, tmp string) error {
	old := ""
	if _, err := os.Lstat(target); err == nil {
		suffix, rerr := randomSuffix()
		if rerr != nil {
			return rerr
		}
		old = target + ".old-" + suffix
		if err := os.Rename(target, old); err != nil {
			return fmt.Errorf("move existing target aside: %w", err)
		}
		defer func() { _ = os.RemoveAll(old) }() // 成功后旧产物在此清除；失败时已换回
	}
	if err := os.Rename(tmp, target); err != nil {
		if old != "" {
			if rerr := os.Rename(old, target); rerr != nil {
				return fmt.Errorf("install new product: %w (also failed to restore previous target: %v)", err, rerr)
			}
		}
		return fmt.Errorf("install new product: %w", err)
	}
	return nil
}

// kernelExcludes 是内核目录里**不原样复制**的文件（相对框架仓根，斜杠路径）：
//   - cmd/cli/main.go 由 RenderCLIMain 单形态裁剪后渲染（去掉脚手架 import/命令）；
//   - cmd/cli/new.go|new_test.go 是框架脚手架命令本身，生成项目不含 tools/generator；
//   - cmd/cli/capability.go|capability_test.go 同理（T7 的 `capability add` 依赖 tools/generator，
//     生成项目既没有该工具树也不该注册这个命令 —— 命令注册由 RenderCLIMain 同步剔除）。
var kernelExcludes = map[string]bool{
	"cmd/cli/main.go":            true,
	"cmd/cli/new.go":             true,
	"cmd/cli/new_test.go":        true,
	"cmd/cli/capability.go":      true,
	"cmd/cli/capability_test.go": true,
	// activecaps_test.go 断言的是**框架**的多形态行为（registry.Lookup("minimal")、框架形态的
	// 声明集与 catalog 拓扑序）。生成项目只有一个形态、声明集也不同，这些断言天然不成立
	// （它编译得过、vet 也过，只是期望值不匹配）。T2 已按单形态重渲染 cmd/cli，故不再复制该测试，
	// 让生成项目的 `go test ./...` 保持全绿。
	"cmd/cli/activecaps_test.go": true,
}

// copyKernel 复制内核必需目录与单文件（S1：内核与形态无关，原样复制；kernelExcludes 除外）。
func copyKernel(root, dst string) error {
	for _, rel := range kernelDirs {
		filter := func(r string, d fs.DirEntry) bool {
			if !skipTransient(r, d) {
				return false
			}
			return d.IsDir() || !kernelExcludes[filepath.ToSlash(filepath.Join(rel, r))]
		}
		if _, _, err := CopyTree(filepath.Join(root, filepath.FromSlash(rel)), filepath.Join(dst, filepath.FromSlash(rel)), filter); err != nil {
			return fmt.Errorf("copy kernel directory %s: %w", rel, err)
		}
	}
	for _, rel := range kernelFiles {
		if err := copyOneFile(root, dst, rel); err != nil {
			return err
		}
	}
	return nil
}

// copyCapabilities 复制「需要完整能力目录」的能力（声明集 ∪ 编译闭包），并按 S4 收窄
// Descriptor.Drivers。注意**不遍历 set.Copy**：Copy 还含 MigrationOnly（S2：migrations +
// domain + 生成的 module.go）与 DomainOnly（裁定④：只带 domain/），那些由
// copyMigrationOnly / copyKernelRequiredDomains 负责，整目录复制会把它们的 wire/cli 一起
// 带进来，与「未选中即不出现」冲突。
// filter 只对**驱动子目录**生效：能力目录的第一段若 ∈ Descriptor.Drivers 且 ∉ 选中集，
// 整棵子树不进（recon §3：复制能力目录会连带拖进未选中驱动）。
func copyCapabilities(root, dst string, set CapabilitySet) error {
	descs := descriptorIndex(catalog.All())
	narrowed := make([]string, 0, len(set.Copy))
	for _, name := range set.Copy {
		if slices.Contains(set.MigrationOnly, name) || slices.Contains(set.DomainOnly, name) {
			continue // 部分携带：内容由 copyMigrationOnly / copyKernelRequiredDomains 决定
		}
		narrowed = append(narrowed, name)
	}
	for _, name := range narrowed {
		src := filepath.Join(root, filepath.FromSlash(capabilityDirPrefix), name)
		selected := set.Drivers[name]
		declared := map[string]bool{}
		for _, drv := range descs[name].Drivers {
			declared[drv] = true
		}
		filter := func(rel string, d fs.DirEntry) bool {
			if !d.IsDir() {
				return true
			}
			first, _, _ := strings.Cut(rel, "/")
			return !declared[first] || slices.Contains(selected, first)
		}
		if _, _, err := CopyTree(src, filepath.Join(dst, filepath.FromSlash(capabilityDirPrefix), name), filter); err != nil {
			return fmt.Errorf("copy capability %s: %w", name, err)
		}
		if err := narrowDrivers(dst, name, set); err != nil {
			return err
		}
	}
	return nil
}

// narrowDrivers 收窄已复制能力里的 Descriptor.Drivers（S4）：生成项目的 check-capabilities
// 断言①要求「声明的每个驱动目录都存在」，驱动目录按选中集过滤后必须同步收窄声明。
func narrowDrivers(dst, name string, set CapabilitySet) error {
	dir := filepath.Join(dst, filepath.FromSlash(capabilityDirPrefix), name)
	file, err := descriptorFile(dir)
	if err != nil {
		return fmt.Errorf("capability %s: %w", name, err)
	}
	if file == "" {
		return nil // 声明了驱动但根包没有 Descriptor 文件的能力不存在；防御性放行。
	}
	if err := patchDrivers(file, set.Drivers[name]); err != nil {
		return err
	}
	return nil
}

// copyMigrationOnly 落地 S2 的「迁移携带」目录：只复制 migrations/{mysql,postgres}/**，再加一份
// 生成的 module.go（//go:embed migrations + Descriptor{Name, Migrations, Owns}）。不含
// wire.go/cli/驱动目录，也不展开其编译闭包（否则 auth 链会被拖进来）。
func copyMigrationOnly(root, dst string, set CapabilitySet) error {
	for _, name := range set.MigrationOnly {
		srcRel := filepath.Join(capabilityDirPrefix, name)
		if _, _, err := CopyTree(filepath.Join(root, filepath.FromSlash(srcRel), "migrations"),
			filepath.Join(dst, filepath.FromSlash(srcRel), "migrations"), nil); err != nil {
			return fmt.Errorf("copy migrations of capability %s: %w", name, err)
		}
		// domain/ 是 internal/app（seed）的**编译期依赖**（叶子包，无能力内部依赖）；
		// application/infrastructure/interfaces/wire/cli 仍不复制（S2 修正）。
		domainSrc := filepath.Join(root, filepath.FromSlash(srcRel), "domain")
		if info, err := os.Stat(domainSrc); err == nil && info.IsDir() {
			if _, _, err := CopyTree(domainSrc, filepath.Join(dst, filepath.FromSlash(srcRel), "domain"), nil); err != nil {
				return fmt.Errorf("copy domain of capability %s: %w", name, err)
			}
		}
		// Owns 必须取自**框架仓**的 Descriptor（生成目录里此刻还没有 module.go）。
		file, err := descriptorFile(filepath.Join(root, filepath.FromSlash(srcRel)))
		if err != nil {
			return fmt.Errorf("capability %s: %w", name, err)
		}
		if file == "" {
			return fmt.Errorf("capability %s has no Descriptor source to carry migrations", name)
		}
		out, err := renderModuleOnly(file, name)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(srcRel), "module.go")
		if err := os.WriteFile(target, out, goFileMode); err != nil {
			return fmt.Errorf("write %s: %w", filepathSlash(target), err)
		}
	}
	return nil
}

// copyKernelRequiredDomains 复制「内核编译期 domain 依赖」（裁定 ④）：只带
// internal/capabilities/<cap>/domain，不带该能力的其它子包。覆盖表见 kernelRequiredDomains。
func copyKernelRequiredDomains(root, dst string, set CapabilitySet) error {
	for _, entry := range kernelRequiredDomains {
		name, _, _ := strings.Cut(entry, "/")
		if !slices.Contains(set.DomainOnly, name) {
			continue
		}
		rel := filepath.Join(capabilityDirPrefix, filepath.FromSlash(entry))
		if _, _, err := CopyTree(filepath.Join(root, rel), filepath.Join(dst, rel), nil); err != nil {
			return fmt.Errorf("copy kernel-required domain %s: %w", entry, err)
		}
	}
	return nil
}

// descriptorIndex 建 能力名 → Descriptor 索引（生成期只读，用于驱动过滤与声明收窄）。
func descriptorIndex(all []contract.Descriptor) map[string]contract.Descriptor {
	out := make(map[string]contract.Descriptor, len(all))
	for _, d := range all {
		out[d.Name] = d
	}
	return out
}

// descriptorFile 在能力根包目录里找含 `var Descriptor` 的 .go 文件（非测试）；没有则返回 ""。
func descriptorFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		if strings.Contains(string(content), "var Descriptor") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", nil
}

// planResult 是 --dry-run 的结果：只统计将要复制/生成的内容，绝不落盘（打印交给 CLI 层，
// 库只返回数据，便于测试断言）。
func planResult(root string, set CapabilitySet, module string, opts NewOptions) (*Result, error) {
	res := &Result{
		Dir:          absPath(opts.Dir),
		Module:       module,
		Shape:        set.Shape,
		Capabilities: slices.Clone(set.Declared),
		CopySet:      slices.Clone(set.Copy),
		Drivers:      flattenDrivers(set.Drivers),
	}
	for _, rel := range kernelDirs {
		count, err := countTree(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		for excluded := range kernelExcludes {
			if strings.HasPrefix(excluded, rel+"/") {
				count--
			}
		}
		res.FileCount += count
	}
	res.FileCount += len(kernelFiles)
	// 内核编译期 domain 依赖（裁定④）：只算 domain/ 子树。
	for _, entry := range kernelRequiredDomains {
		name, _, _ := strings.Cut(entry, "/")
		if !slices.Contains(set.DomainOnly, name) {
			continue
		}
		count, err := countTree(filepath.Join(root, filepath.FromSlash(capabilityDirPrefix), filepath.FromSlash(entry)))
		if err != nil {
			return nil, err
		}
		res.FileCount += count
	}
	// 能力目录：完整复制的按实际文件数统计（含驱动过滤），迁移携带的只算 migrations + 生成的 module.go。
	descs := descriptorIndex(catalog.All())
	for _, name := range set.Copy {
		src := filepath.Join(root, filepath.FromSlash(capabilityDirPrefix), name)
		if slices.Contains(set.DomainOnly, name) {
			continue // 已在上面按 domain/ 单独统计（裁定④：只带 domain/）
		}
		if slices.Contains(set.MigrationOnly, name) {
			count, err := countTree(filepath.Join(src, "migrations"))
			if err != nil {
				return nil, err
			}
			if info, serr := os.Stat(filepath.Join(src, "domain")); serr == nil && info.IsDir() {
				domainCount, derr := countTree(filepath.Join(src, "domain"))
				if derr != nil {
					return nil, derr
				}
				count += domainCount
			}
			res.FileCount += count + 1
			continue
		}
		declared := map[string]bool{}
		for _, drv := range descs[name].Drivers {
			declared[drv] = true
		}
		selected := set.Drivers[name]
		count := 0
		err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				rel, rerr := filepath.Rel(src, p)
				if rerr != nil {
					return rerr
				}
				rel = filepath.ToSlash(rel)
				if rel != "." {
					first, _, _ := strings.Cut(rel, "/")
					if declared[first] && !slices.Contains(selected, first) {
						return fs.SkipDir
					}
				}
				return nil
			}
			if d.Type()&fs.ModeSymlink == 0 {
				count++
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		res.FileCount += count
	}
	res.FileCount += len(shapeFiles(set))
	res.FileCount += len(catalogFiles())
	// T6：资产（deploy/** + docs/openapi）。用与 CopyAssets 同一个 assetFiles 口径逐文件计数，
	// 重叠前缀（deploy/k8s 与 deploy/k8s/openobserve.yaml）不会重复计入。
	assets, err := AssetsFor(set)
	if err != nil {
		return nil, err
	}
	res.Assets = assets
	assetList, err := assetFiles(root, assets)
	if err != nil {
		return nil, err
	}
	res.FileCount += len(assetList)
	res.FileCount += len(configFiles())
	res.FileCount++ // cmd/cli/main.go（渲染）
	// T5：tools 树（与 copyToolsAndScripts 同一过滤口径，见 countTreeFiltered）与三份构建文件。
	for _, rel := range copyTools {
		count, err := countTreeFiltered(filepath.Join(root, filepath.FromSlash(rel)), func(_ string, d fs.DirEntry) bool {
			return d.IsDir() || !skipToolTestFile(d.Name())
		})
		if err != nil {
			return nil, err
		}
		res.FileCount += count
	}
	res.FileCount += len(buildFiles())
	return res, nil
}

// buildFiles 返回 renderBuildFiles 渲染的三份构建文件（dry-run 计数与落地同源）。
func buildFiles() []string {
	return []string{"Makefile", "Dockerfile", "scripts/check_profiles.sh"}
}

// countTreeFiltered 与 countTree 相同，但允许在 skipTransient 之外再加一层过滤（tools 复制集
// 不复制框架侧 _test.go）。口径必须与 CopyTree 的 filter 逐条一致，否则 dry-run 不再是上界。
func countTreeFiltered(root string, filter func(rel string, d fs.DirEntry) bool) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel != "." && !skipTransient(rel, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if rel != "." && filter != nil && !filter(rel, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() && d.Type()&fs.ModeSymlink == 0 {
			count++
		}
		return nil
	})
	return count, err
}

// countTree 只统计目录树里的文件数（不落盘，供 --dry-run 使用）。必须与 CopyTree 用同一个
// skipTransient 口径：否则 .DS_Store 这类被复制跳过的文件会被 dry-run 计入，预估值虚高。
func countTree(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		if rel != "." && !skipTransient(filepath.ToSlash(rel), d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() && d.Type()&fs.ModeSymlink == 0 {
			count++
		}
		return nil
	})
	return count, err
}

// formatTree 对生成树里的 .go 文件跑 gofmt（RewriteModule 已格式化被改写的文件，但
// registry/assembly/drivers/active 是重写之后才渲染的）。
func formatTree(dst string) error {
	return filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		formatted, err := format.Source(content)
		if err != nil {
			return fmt.Errorf("gofmt %s: %w", relPath(dst, p), err)
		}
		if string(formatted) == string(content) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(p, formatted, info.Mode().Perm())
	})
}

// copyOneFile 复制单个文件（保持相对路径与权限位）。
func copyOneFile(root, dst, rel string) error {
	src := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("kernel file %s: %w", rel, err)
	}
	if info.IsDir() {
		return fmt.Errorf("kernel file %s is a directory", rel)
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read kernel file %s: %w", rel, err)
	}
	target := filepath.Join(dst, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	return os.WriteFile(target, content, info.Mode().Perm())
}

// skipTransient 过滤掉不该进生成项目的临时/无关条目（复制内核目录与 assets 共用）。
func skipTransient(rel string, d fs.DirEntry) bool {
	base := d.Name()
	switch base {
	case ".git", ".DS_Store", ".idea", ".vscode":
		return false
	}
	if strings.HasSuffix(base, ".tmp-") || strings.HasPrefix(base, ".") && strings.Contains(base, ".tmp-") {
		return false
	}
	return true
}

// flattenDrivers 把 能力→驱动 映射拍平成 `cap:drv` 列表（报告与 marker 用，顺序确定）。
func flattenDrivers(drivers map[string][]string) []string {
	out := make([]string, 0, len(drivers))
	for name, list := range drivers {
		for _, drv := range list {
			out = append(out, name+":"+drv)
		}
	}
	sort.Strings(out)
	return out
}

// randomSuffix 生成 <dir>.tmp-<rand> / <dir>.old-<rand> 的随机后缀。
func randomSuffix() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()), nil
	}
	return hex.EncodeToString(buf[:]), nil
}

// absPath 返回绝对路径；无法解析时原样返回（调用方随后会因路径不存在而报错）。
func absPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// filepathSlash 把路径转成斜杠形式（错误文案与报告统一口径）。
func filepathSlash(p string) string { return filepath.ToSlash(p) }
