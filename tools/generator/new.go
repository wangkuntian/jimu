package generator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
)

// NewOptions 是 `jimu new` 的全部输入（CLI 与测试共用同一结构，参数校验只在
// ParseCapabilitySet 一处）。
type NewOptions struct {
	Dir     string // 目标目录
	Profile string // --profile：形态名（与 With 互斥）
	With    string // --with：<cap>[:<drv>][,...]（与 Profile 互斥）
	Shape   string // --shape：--with 时的形态名，默认 app
	Module  string // --module：模块路径，默认由 Dir 推导

	NoTidy bool // 跳过 go mod tidy（T8 接入）
	DryRun bool // 只打印计划，不落盘
	Force  bool // 只覆盖带 .jimu-generated 标记的既有产物
	Report bool // 额外写 docs/profiles/generated-report.md（T8 接入）
}

// Result 是一次生成的摘要（--dry-run 与 --report 共用）。
type Result struct {
	Dir          string
	Module       string
	Shape        string
	Capabilities []string
	Drivers      []string
	Assets       []string
	Files        []string
	FileCount    int
	Lines        int
}

// markerFile 是生成器产物的标记（dot 文件不参与 `grep -rn '"jimu/'`）：--force 的识别依据，
// 也是 `jimu capability add` 的输入（S7/S8）。
const markerFile = ".jimu-generated"

// marker 是 markerFile 的 JSON 结构（S7 规定字段）。
type marker struct {
	Generator      string              `json:"generator"`
	Version        string              `json:"version"`
	SourceRoot     string              `json:"sourceRoot"`
	SourceCommit   string              `json:"sourceCommit"`
	Module         string              `json:"module"`
	Shape          string              `json:"shape"`
	Profile        string              `json:"profile"`
	Capabilities   []string            `json:"capabilities"`
	DomainOnly     []string            `json:"domainOnly"`
	Drivers        map[string][]string `json:"drivers"`
	Assets         []string            `json:"assets"`
	Files          []string            `json:"files"`
	DiscardedTests []string            `json:"discardedTests"`
}

// generatorVersion 是写入标记的生成器版本（与本仓版本解耦，仅供 add/upgrade 判断口径）。
const generatorVersion = "p2.7"

// NewProject 生成项目。执行顺序与回滚语义见计划第 2 节裁定 7/8：
//
//	① 解析能力集 → ② 复制内核必需目录 → ③ 按能力复制（含驱动过滤）
//	→ ④ 渲染 registry/profiles/<shape>/active（catalog/app.yaml 由 T3/T4 补齐）
//	→ ⑦ module 受控重写 + gofmt → ⑧ go mod tidy（--no-tidy 关闭）→ ⑨ 自检（T8）
//
// 产物先写 <dir>.tmp-<rand>，成功后原子 rename；任何一步失败都删除临时目录（defer RemoveAll），
// 绝不留半成品（验收⑤）。--force 时先把既有产物 rename 成 <dir>.old-<rand>，rename 成功后再删，
// 避免「先删后建」留下空目录窗口。
func NewProject(opts NewOptions) (*Result, error) {
	set, err := ParseCapabilitySet(opts.Profile, opts.With, opts.Shape)
	if err != nil {
		return nil, err
	}
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
	if opts.Report {
		// T2 阶段明确报错而不是静默 no-op（报告落点归 T8）。
		return nil, fmt.Errorf("--report 将在 T8 实现（not implemented yet）")
	}
	target := absPath(opts.Dir)
	if opts.DryRun {
		return planResult(root, set, module, opts)
	}
	if err := preflightTarget(target, opts.Force); err != nil {
		return nil, err
	}
	parent := filepath.Dir(target)
	tmp, err := os.MkdirTemp(parent, filepath.Base(target)+".tmp-")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }() // rename 成功后 tmp 已不存在，幂等
	if err := generateInto(root, tmp, set, module, opts); err != nil {
		return nil, err
	}
	if err := swapIntoPlace(target, tmp); err != nil {
		return nil, err
	}
	return collectResult(target, set, module, root)
}

// generateInto 把全部产物写进临时目录 tmp（此时还没有任何 rename，失败由调用方删 tmp）。
func generateInto(root, dst string, set CapabilitySet, module string, opts NewOptions) error {
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
	discarded, err := pruneUnsatisfiableTests(dst, module)
	if err != nil {
		return err
	}
	if err := formatTree(dst); err != nil {
		return err
	}
	return writeMarker(root, dst, set, module, assets, discarded)
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

// pruneUnsatisfiableTests 按「逐文件 import 可满足性」裁剪生成项目的测试树：
//
//	生产文件不满足 → 报错（这是 C1「复制闭包不完整」的捕获网，宁可失败也不产出编译不过的项目）；
//	测试文件不满足 → **该目录下的测试文件整组丢弃**，并把清单记入 .jimu-generated 的 discardedTests。
//
// 为什么是「整组」而不是「逐文件」：同一 package 的 `_test.go` 之间会互相引用（e2e 的
// helpers_test.go 提供 newTestAppWithDB，被同目录多个测试用），只删 import 不满足的那个文件会
// 留下「undefined: xxx」——`go vet`/`go test` 仍编译不过。同目录测试整组保留或整组丢弃，
// 既满足「所有 jimu/... import 都能被满足」，又不依赖符号级分析。
//
// 只删测试、**不扩复制集**：未选中能力不会因为某个测试 import 了它而被拉进生成项目。
func pruneUnsatisfiableTests(dst, module string) ([]string, error) {
	pkgs := map[string]bool{}
	testFiles := map[string][]string{} // 目录 → 该目录下的 _test.go
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
		dir := filepath.ToSlash(filepath.Dir(rel))
		if strings.HasSuffix(d.Name(), "_test.go") {
			testFiles[dir] = append(testFiles[dir], rel)
			return nil
		}
		pkgs[dir] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	prefix := module + "/"
	var discarded []string
	for _, dir := range slices.Sorted(maps.Keys(testFiles)) {
		dropDir := false
		for _, rel := range testFiles[dir] {
			missing, err := missingModuleImports(filepath.Join(dst, filepath.FromSlash(rel)), prefix, pkgs)
			if err != nil {
				return nil, err
			}
			if len(missing) > 0 {
				dropDir = true
				break
			}
		}
		if !dropDir {
			continue
		}
		for _, rel := range testFiles[dir] {
			if rerr := os.Remove(filepath.Join(dst, filepath.FromSlash(rel))); rerr != nil {
				return nil, fmt.Errorf("remove unsatisfiable test %s: %w", rel, rerr)
			}
			discarded = append(discarded, rel)
		}
	}
	// 生产文件的缺口单独再扫一遍（错误要精确到文件，不能被上面的整组逻辑吞掉）。
	err = filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
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
		rel := relPath(dst, p)
		missing, merr := missingModuleImports(p, prefix, pkgs)
		if merr != nil {
			return merr
		}
		if len(missing) > 0 {
			return fmt.Errorf("复制集缺口：生产文件 %s import %s，但生成树里没有该包", rel, strings.Join(missing, ", "))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(discarded)
	return discarded, nil
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
//   - 目标不存在、或存在但为空目录 → 放行；
//   - 目标非空且无 --force → 报错 not empty；
//   - 目标非空且带 .jimu-generated 标记 → 放行（由 swapIntoPlace 整体替换）。
func preflightTarget(target string, force bool) error {
	if _, err := os.ReadDir(target); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect target %s: %w", filepathSlash(target), err)
	}
	// 既有目录的判据是「是不是本生成器的产物」：没有标记一律拒绝（即使带了 --force ——
	// --force 的语义是「覆盖生成器产物」，不是「强行写任何目录」）。
	if _, err := os.Stat(filepath.Join(target, markerFile)); err != nil {
		return fmt.Errorf("target directory %s is not empty and has no %s marker; --force only overwrites generator products", filepathSlash(target), markerFile)
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
//   - cmd/cli/new.go|new_test.go 是框架脚手架命令本身，生成项目不含 tools/generator。
var kernelExcludes = map[string]bool{
	"cmd/cli/main.go":     true,
	"cmd/cli/new.go":      true,
	"cmd/cli/new_test.go": true,
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
		Capabilities: slices.Clone(set.Copy),
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

// collectResult 在产物就位后统计文件数/行数与文件清单（--report 的输入）。
func collectResult(target string, set CapabilitySet, module, root string) (*Result, error) {
	res := &Result{
		Dir:          target,
		Module:       module,
		Shape:        set.Shape,
		Capabilities: slices.Clone(set.Copy),
		Drivers:      flattenDrivers(set.Drivers),
	}
	assets, err := AssetsFor(set)
	if err != nil {
		return nil, err
	}
	res.Assets = assets
	err = filepath.WalkDir(target, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		rel := relPath(target, p)
		if rel == markerFile {
			return nil // 标记不是产物文件（marker.files 也不含它）
		}
		res.Files = append(res.Files, rel)
		res.FileCount++
		if strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, ".yaml") || strings.HasSuffix(rel, ".yml") ||
			strings.HasSuffix(rel, ".sh") || strings.HasSuffix(rel, ".mod") {
			content, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			res.Lines += lineCount(string(content))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(res.Files)
	return res, nil
}

// lineCount 统计文本行数（无尾换行的最后一行也算一行）。
func lineCount(text string) int {
	if text == "" {
		return 0
	}
	n := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}

// writeMarker 写 .jimu-generated（S7/S8 + Minor 10）：--force 的识别依据、capability add 与
// T8 report 的输入。`files` 是本次落地文件的完整清单（排序），`assets` 是本次复制的资产路径
// （AssetsFor 的前缀口径：`deploy/helm`、`docs/openapi`…），`discardedTests` 是被裁剪掉的测试文件。
func writeMarker(root, dst string, set CapabilitySet, module string, assets, discarded []string) error {
	m := marker{
		Generator:      "jimu new",
		Version:        generatorVersion,
		SourceRoot:     absPath(root),
		SourceCommit:   gitCommit(root),
		Module:         module,
		Shape:          set.Shape,
		Profile:        set.Profile,
		Capabilities:   slices.Clone(set.Declared),
		DomainOnly:     append([]string{}, set.DomainOnly...),
		Drivers:        set.Drivers,
		Assets:         assets,
		Files:          []string{},
		DiscardedTests: append([]string{}, discarded...),
	}
	if m.Assets == nil {
		m.Assets = []string{}
	}
	if m.Drivers == nil {
		m.Drivers = map[string][]string{}
	}
	if err := filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := relPath(dst, p)
		if rel != markerFile {
			m.Files = append(m.Files, rel)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("collect generated files: %w", err)
	}
	sort.Strings(m.Files)
	content, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", markerFile, err)
	}
	return os.WriteFile(filepath.Join(dst, markerFile), append(content, '\n'), 0o644)
}

// gitCommit 取框架仓当前提交；取不到（无 git / 无提交 / 超时）返回空串，不阻断生成。
func gitCommit(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
