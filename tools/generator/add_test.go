package generator

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/catalog"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件是 P2.7 Task 7（`jimu capability add <name>`）的落地验收。四条硬输入各由一条测试钉死：
//
//	① 重渲染必须重算（裁定 ⑬/T3 裁定 7）：TestAddCapabilityKeepsMigrationCarryAndKnownNames
//	   —— add 后 tenant 仍在 entries、knownNames 仍 25、catalog 拓扑序正确；
//	② 配置段合并以生成项目现有 app.yaml 为底（裁定 ⑬）：
//	   TestAddCapabilityPreservesEditedConfigSections —— 手改值仍在 + 新段出现 + 二次 add 幂等；
//	③ 资产按前缀口径重派生（T6 审查）：
//	   TestAddCapabilityCarriesNewCapabilityAssets —— apidocs 的 docs/openapi 被复制、marker 前缀更新；
//	④ 不做依赖闭包自动补全：TestAddCapabilityRejectsMissingHardDependency。

// generateForTest 在 t.TempDir() 里生成一个项目并返回目录（module 默认 example.com/proj）。
func generateForTest(t *testing.T, opts NewOptions) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "proj")
	opts.Dir = dir
	if opts.Module == "" {
		opts.Module = "example.com/proj"
	}
	opts.NoTidy = true
	_, err := newProjectForTest(t, opts)
	require.NoError(t, err)
	return dir
}

// TestAddCapabilityRefreshesGeneratedReport `capability add` 改变了能力集与文件数，项目里**已有**的
// `--report` 产物必须同批刷新（否则它静默过期：旧装配集、旧文件数）；本来没有报告的项目不得凭空多出一份。
func TestAddCapabilityRefreshesGeneratedReport(t *testing.T) {
	// 无报告的对照项目先生成（NewProject 需要 cwd 在框架仓内；下面会 t.Chdir 到生成项目）。
	plain := generateForTest(t, NewOptions{Profile: "minimal"})

	dir := generateForTest(t, NewOptions{Profile: "minimal", Report: true})
	path := filepath.Join(dir, filepath.FromSlash(reportRelPath))
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(before), "dataops")

	// 关键：`capability add` 的 cwd 就是生成项目（文档用法 `cd proj && jimu capability add x`）——
	// 框架源根只能取自 marker.SourceRoot，不能按 cwd 重新发现（Fix round 2 修掉的 bug）。
	t.Chdir(dir)

	_, err = AddCapability(AddOptions{Name: "dataops", Dir: dir})
	require.NoError(t, err)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(after), "dataops", "add 后报告必须反映新的装配集")
	assert.NotEqual(t, string(before), string(after), "报告内容必须真的重算")
	assert.Greater(t, reportGeneratedCount(t, string(after)), reportGeneratedCount(t, string(before)),
		"add 复制了新能力目录，报告的「生成文件数」必须随之更新")

	// 重跑一次：报告内容稳定（幂等），且数字继续跟随新的复制集。
	_, err = AddCapability(AddOptions{Name: "storage", Dir: dir})
	require.NoError(t, err)
	again, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(again), "storage")
	assert.Contains(t, string(again), "dataops")
	assert.Greater(t, reportGeneratedCount(t, string(again)), reportGeneratedCount(t, string(after)))

	// 没有报告的项目：add 不得创建报告（报告只在 --report 时产出）。
	_, err = AddCapability(AddOptions{Name: "dataops", Dir: plain})
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(plain, filepath.FromSlash(reportRelPath)))
}

// reportGeneratedCount 从报告文本里取出「生成文件数」的实测值（解析失败即 fail）。
func reportGeneratedCount(t *testing.T, report string) int {
	t.Helper()
	m := regexp.MustCompile(`(?m)^\| 生成文件数 \| (\d+) \|$`).FindStringSubmatch(report)
	require.Len(t, m, 2, "报告里必须有「生成文件数」行")
	n, err := strconv.Atoi(m[1])
	require.NoError(t, err)
	return n
}

// hashTree 返回整棵树的「相对路径 + 内容 sha256」摘要：用于断言幂等与失败回滚（只比内容，不比 mtime）。
func hashTree(t *testing.T, dir string) string {
	t.Helper()
	sums := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		content, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		sums[relPath(dir, p)] = fmt.Sprintf("%x", sha256.Sum256(content))
		return nil
	}))
	var b strings.Builder
	for _, name := range slices.Sorted(maps.Keys(sums)) {
		fmt.Fprintf(&b, "%s %s\n", name, sums[name])
	}
	return b.String()
}

// catalogVarElts 取生成版 catalog.go 里某个包级 var 的复合字面量元素。
func catalogVarElts(t *testing.T, dir, name string) []ast.Expr {
	t.Helper()
	path := filepath.Join(dir, "internal", "capabilities", "catalog", "catalog.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != name || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			return lit.Elts
		}
	}
	t.Fatalf("生成版 catalog.go 里没有 var %s", name)
	return nil
}

// catalogEntryNames 返回生成版 catalog.entries 的能力名（顺序即拓扑序）。
func catalogEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	for _, elt := range catalogVarElts(t, dir, "entries") {
		sel, ok := elt.(*ast.SelectorExpr)
		require.Truef(t, ok, "entries 元素应为 <alias>.Descriptor，实际 %T", elt)
		id, ok := sel.X.(*ast.Ident)
		require.Truef(t, ok, "entries 元素别名应为标识符，实际 %T", sel.X)
		out = append(out, strings.TrimSuffix(id.Name, "module"))
	}
	return out
}

// catalogKnownNames 返回生成版 catalog.knownNames（框架全量能力名白名单）。
func catalogKnownNames(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	for _, elt := range catalogVarElts(t, dir, "knownNames") {
		lit, ok := elt.(*ast.BasicLit)
		require.Truef(t, ok, "knownNames 元素应为字符串字面量，实际 %T", elt)
		v, err := strconv.Unquote(lit.Value)
		require.NoError(t, err)
		out = append(out, v)
	}
	return out
}

// readProjectFile 读生成项目里的文件（相对路径）。
func readProjectFile(t *testing.T, dir, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return string(content)
}

// writeProjectFile 写生成项目里的文件（相对路径）。
func writeProjectFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// noStagingResidue 断言目录的父目录里没有 .tmp-/.bak-/.old- 残留（原子性的自查）。
func noStagingResidue(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp-")
		assert.NotContains(t, e.Name(), ".bak-")
		assert.NotContains(t, e.Name(), ".old-")
	}
}

// TestAddCapabilityKeepsMigrationCarryAndKnownNames 是硬输入 ① 的钉死测试（T3 裁定 7 / 裁定 ⑬）：
// marker.capabilities 是**声明集**（不含迁移携带的 tenant），add 时必须经 CapabilityRoots 重算 ——
// 直接把 marker 集合喂渲染器会让 tenant 掉出 entries、knownNames 变空，生成项目少建表/少加列且门禁红。
func TestAddCapabilityKeepsMigrationCarryAndKnownNames(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})

	// 前置：minimal 的 tenant 是迁移携带（不在声明集里，但在 entries 里）。
	before, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.NotContains(t, before.Capabilities, "tenant", "marker.capabilities 是声明集，不含迁移携带的 tenant")
	assert.Contains(t, catalogEntryNames(t, dir), "tenant", "minimal 的 tenant 必须随迁移进 entries")
	require.Len(t, catalogKnownNames(t, dir), 25)

	_, err = AddCapability(AddOptions{Dir: dir, Name: "dataops", From: root})
	require.NoError(t, err)

	after, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.Contains(t, after.Capabilities, "dataops")
	assert.NotContains(t, after.Capabilities, "tenant", "add 不得把迁移携带能力写进声明集")

	entries := catalogEntryNames(t, dir)
	assert.Contains(t, entries, "dataops", "新增能力必须进 entries")
	assert.Contains(t, entries, "tenant", "迁移携带能力必须仍在 entries（否则 jimu migrate 漏建表）")
	// catalog 拓扑序：user < access < tenant < auth < dataops（catalog 的顺序），Ungated 追加在尾部。
	assert.Equal(t, []string{"user", "access", "tenant", "auth", "dataops", "encryption", "notification"}, entries)

	known := catalogKnownNames(t, dir)
	assert.Len(t, known, 25, "knownNames 恒为框架全量能力名（catalog 18 ∪ Ungated 7）")
	assert.Contains(t, known, "dataops")
	assert.Contains(t, known, "tenant")
}

// TestAddCapabilityPreservesEditedConfigSectionsAndIsIdempotent 是硬输入 ② 的钉死测试（裁定 ⑬）：
// RenderAppConfig 是**纯过滤**（以蓝本为 src），add 不得用它重渲染整份配置 —— 必须以生成项目现有
// app.yaml 为底、用蓝本补齐新增段，否则用户手改的段值会被静默覆盖。
func TestAddCapabilityPreservesEditedConfigSectionsAndIsIdempotent(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})

	const appRel = "configs/app.yaml"
	const prodRel = "configs/app.prod.yaml"
	const valuesRel = "deploy/helm/values.yaml"

	// 手改：改两个已存在段的值，并加一个用户自有段。
	app := readProjectFile(t, dir, appRel)
	require.Contains(t, app, "timeout_sec: 60")
	app = strings.Replace(app, "  timeout_sec: 60", "  timeout_sec: 12345", 1)
	app += "\n# 用户自有段（生成器不认识，合并时不得丢失）\nmycustom:\n  flag: true\n"
	writeProjectFile(t, dir, appRel, app)

	prod := readProjectFile(t, dir, prodRel)
	require.Contains(t, prod, "  timeout_sec: 30          # 请求超时秒数")
	writeProjectFile(t, dir, prodRel, strings.Replace(prod, "  timeout_sec: 30          # 请求超时秒数", "  timeout_sec: 12345", 1))

	values := readProjectFile(t, dir, valuesRel)
	require.Contains(t, values, "replicaCount: 2")
	writeProjectFile(t, dir, valuesRel, strings.Replace(values, "replicaCount: 2", "replicaCount: 3", 1))

	// audit ∉ minimal 的 entries（既未声明也未被迁移携带）→ 允许 add；它自带 configs 段与 values 同名键。
	res, err := AddCapability(AddOptions{Dir: dir, Name: "audit", From: root})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Changed)

	after := readProjectFile(t, dir, appRel)
	assert.Contains(t, after, "timeout_sec: 12345", "手改的段值必须保留")
	assert.Contains(t, after, "mycustom:", "用户自有段不得被合并丢弃")
	assert.Contains(t, after, "audit:", "新增能力的段必须出现")
	assert.NotContains(t, after, "\nqueue:", "未选中能力的段仍不出现")

	afterProd := readProjectFile(t, dir, prodRel)
	assert.Contains(t, afterProd, "timeout_sec: 12345", "app.prod.yaml 同口径")
	assert.Contains(t, afterProd, "audit:")

	afterValues := readProjectFile(t, dir, valuesRel)
	assert.Contains(t, afterValues, "replicaCount: 3", "values.yaml 的手改值必须保留")
	assert.Contains(t, afterValues, "audit:", "新增能力的同名顶层键必须出现")

	// 二次 add（--force 覆盖同一能力）幂等：changed 为空且树逐字节不变。
	before := hashTree(t, dir)
	res2, err := AddCapability(AddOptions{Dir: dir, Name: "audit", From: root, Force: true})
	require.NoError(t, err)
	assert.Empty(t, res2.Changed, "二次 add 必须无改动")
	assert.Equal(t, before, hashTree(t, dir))
	noStagingResidue(t, dir)
}

// TestAddCapabilityRejectsMissingHardDependency 是硬输入 ④：缺 Requires 一律报错并提示先加依赖，
// **绝不**自动补全闭包（否则用户以为只加了一个能力，实际装配集被悄悄扩大）。
func TestAddCapabilityRejectsMissingHardDependency(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{With: "user", Shape: "app"})
	before := hashTree(t, dir)

	_, err := AddCapability(AddOptions{Dir: dir, Name: "passkey", From: root})
	require.ErrorContains(t, err, `requires "auth"`)
	require.ErrorContains(t, err, "add it first")
	assert.Equal(t, before, hashTree(t, dir), "校验失败不得留下半成品")
	noStagingResidue(t, dir)
}

// TestAddCapabilityExistingMessagesDistinguishMigrationCarry 是 Minor ① 的文案护栏：已声明的能力
// 与「只随迁移携带」的能力必须给出不同指引 —— 后者在生成项目里没有实现，--force 的语义是「提升为
// 完整声明能力」，不能与普通重复声明共用一句「pass --force to rebuild it」。
func TestAddCapabilityExistingMessagesDistinguishMigrationCarry(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})

	_, err := AddCapability(AddOptions{Dir: dir, Name: "tenant", From: root})
	require.ErrorContains(t, err, "already present")
	require.ErrorContains(t, err, "only as a migration carry")
	require.ErrorContains(t, err, "promote it to a full declared capability")

	_, err = AddCapability(AddOptions{Dir: dir, Name: "user", From: root})
	require.ErrorContains(t, err, "already present")
	require.ErrorContains(t, err, "as a declared capability")
	require.NotContains(t, err.Error(), "migration carry")
}

// TestAddCapabilityRejectsExistingCapability 钉住「能力已在 entries 里 → 报错；--force 才覆盖」，
// 且 --force 的重建是幂等的（内容由声明集纯函数决定）。
func TestAddCapabilityRejectsExistingCapability(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	before := hashTree(t, dir)

	_, err := AddCapability(AddOptions{Dir: dir, Name: "user", From: root})
	require.ErrorContains(t, err, "already present")
	assert.Equal(t, before, hashTree(t, dir), "already present 时不得落盘")

	beforeForce := hashTree(t, dir)
	res, err := AddCapability(AddOptions{Dir: dir, Name: "user", From: root, Force: true})
	require.NoError(t, err)
	assert.Empty(t, res.Changed)
	assert.Equal(t, beforeForce, hashTree(t, dir), "--force 重建同一集合必须逐字节等价（幂等）")
}

// TestAddCapabilityCarriesNewCapabilityAssets 是硬输入 ③：marker.assets 是**前缀清单**，
// add 后必须经 AssetsFor/assetFiles 重派生并把新增能力的资产复制进 deploy/**（apidocs → docs/openapi）。
func TestAddCapabilityCarriesNewCapabilityAssets(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	before, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.NotContains(t, before.Assets, "docs/openapi")

	_, err = AddCapability(AddOptions{Dir: dir, Name: "apidocs", From: root})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(dir, "docs/openapi/swagger.json"))
	after, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.Contains(t, after.Assets, "docs/openapi", "marker.assets 必须按前缀口径更新")
	assert.Contains(t, after.Capabilities, "apidocs")
	assert.Contains(t, readProjectFile(t, dir, "Dockerfile"), "docs/openapi", "构建文件必须同步（未选中资产不出现）")
}

// TestAddCapabilitySelectsFirstDriverAndFiltersDirectories 钉住 S4 + 驱动目录过滤在 add 上同样成立：
// 新增能力的驱动默认取 Descriptor.Drivers 首项，未选中驱动目录不进生成树、声明同步收窄。
func TestAddCapabilitySelectsFirstDriverAndFiltersDirectories(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})

	_, err := AddCapability(AddOptions{Dir: dir, Name: "queue", From: root})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(dir, "internal/capabilities/queue/redis/redis_queue.go"))
	for _, drv := range []string{"kafka", "rabbitmq"} {
		_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities/queue", drv))
		assert.True(t, os.IsNotExist(statErr), "未选中驱动 queue/%s 不应出现", drv)
	}
	m, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"redis"}, m.Drivers["queue"])
	// catalog 拓扑序：queue 在 catalog 里位于 auth 之后（由 catalog 决定，不是追加到尾部）。
	entries := catalogEntryNames(t, dir)
	assert.Contains(t, entries, "queue")
	assert.Less(t, slices.Index(entries, "auth"), slices.Index(entries, "queue"))
	assert.Less(t, slices.Index(entries, "user"), slices.Index(entries, "auth"))
}

// TestAddCapabilityDryRunWritesNothing 钉住 --dry-run：只报计划、绝不落盘、不留暂存目录。
func TestAddCapabilityDryRunWritesNothing(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	before := hashTree(t, dir)

	res, err := AddCapability(AddOptions{Dir: dir, Name: "dataops", From: root, DryRun: true})
	require.NoError(t, err)
	assert.Contains(t, res.Changed, "internal/capabilities/catalog/catalog.go")
	assert.Contains(t, res.Changed, "internal/profiles/minimal/assembly.go")
	assert.Equal(t, before, hashTree(t, dir), "--dry-run 绝不落盘")
	noStagingResidue(t, dir)
}

// TestAddCapabilityRollsBackPartialWrites 钉住原子性：任一文件写失败必须回滚到调用前（内容级）。
func TestAddCapabilityRollsBackPartialWrites(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	before := hashTree(t, dir)

	original := installFile
	t.Cleanup(func() { installFile = original })
	calls := 0
	installFile = func(src, dst, rel string) error {
		calls++
		if calls == 2 {
			return errors.New("boom: injected write failure")
		}
		return original(src, dst, rel)
	}

	_, err := AddCapability(AddOptions{Dir: dir, Name: "dataops", From: root})
	require.ErrorContains(t, err, "boom")
	assert.Equal(t, before, hashTree(t, dir), "失败必须回滚到调用前")
	noStagingResidue(t, dir)
}

// TestAddCapabilityPreservesUserFiles 钉住「只覆盖生成器产物」：用户自有文件既不进 marker.files
// （否则下一次 add 会把它当成生成产物删掉），也不被 gofmt/测试裁剪管线碰到。
func TestAddCapabilityPreservesUserFiles(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	writeProjectFile(t, dir, "NOTES.md", "user notes\n")
	writeProjectFile(t, dir, "myapp/main.go", "package main\n\nfunc main() {}\n")

	_, err := AddCapability(AddOptions{Dir: dir, Name: "dataops", From: root})
	require.NoError(t, err)
	assert.Equal(t, "user notes\n", readProjectFile(t, dir, "NOTES.md"))
	assert.Contains(t, readProjectFile(t, dir, "myapp/main.go"), "func main()")

	m, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.NotContains(t, m.Files, "NOTES.md", "用户文件不得进 marker.files")
	assert.NotContains(t, m.Files, "myapp/main.go")

	// 第二次 add（另一个能力）不得把用户文件当成生成产物清掉。
	_, err = AddCapability(AddOptions{Dir: dir, Name: "search", From: root})
	require.NoError(t, err)
	assert.Equal(t, "user notes\n", readProjectFile(t, dir, "NOTES.md"))
	assert.Contains(t, readProjectFile(t, dir, "myapp/main.go"), "func main()")
}

// TestAddCapabilityRejectsForeignDirectory fail-closed：没有 .jimu-generated 的目录一律拒绝
// （绝不猜模块路径、绝不在任意目录里重渲染）。
func TestAddCapabilityRejectsForeignDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := AddCapability(AddOptions{Dir: dir, Name: "dataops", From: frameworkRootForTest(t)})
	require.ErrorContains(t, err, markerFile)
}

// TestAddCapabilityRejectsUnknownCapability 能力名必须在框架全量集合里（catalog 18 ∪ Ungated 7）。
func TestAddCapabilityRejectsUnknownCapability(t *testing.T) {
	root := frameworkRootForTest(t)
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	_, err := AddCapability(AddOptions{Dir: dir, Name: "ghost", From: root})
	require.ErrorContains(t, err, `unknown capability "ghost"`)
}

// TestMarkerSaveLoadRoundTrip 钉住 marker 的读写接口（S7）：Save 后 Load 逐字段相等。
func TestMarkerSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := &Marker{
		Generator:    "jimu new",
		Version:      generatorVersion,
		SourceRoot:   frameworkRootForTest(t),
		SourceCommit: "deadbeef",
		Module:       "example.com/proj",
		Shape:        "minimal",
		Profile:      "minimal",
		Capabilities: []string{"user", "access"},
		DomainOnly:   []string{"tenant"},
		Drivers:      map[string][]string{"queue": {"redis"}},
		Assets:       []string{"deploy/k8s"},
		Files:        []string{"go.mod"},
	}
	require.NoError(t, want.Save(dir))
	got, err := LoadMarker(dir)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestGeneratedProjectHasNoCapabilityScaffolding 钉住脚手架命令不进生成项目：
// cmd/cli/capability.go 依赖 tools/generator（生成项目不含），既不能复制也不能注册。
func TestGeneratedProjectHasNoCapabilityScaffolding(t *testing.T) {
	dir := generateForTest(t, NewOptions{Profile: "minimal"})
	assert.NoFileExists(t, filepath.Join(dir, "cmd/cli/capability.go"))
	assert.NoFileExists(t, filepath.Join(dir, "cmd/cli/capability_test.go"))
	_, commands := parseCLIMain(t, filepath.Join(dir, "cmd/cli/main.go"))
	assert.NotContains(t, commands, "capability")
	assert.NotContains(t, readProjectFile(t, dir, "cmd/cli/main.go"), "capabilityCmd")
}

// TestMergeSectionBlocksKeepsBaseAndAppendsFresh 直接钉住裁定 ⑬ 的合并原语（不经过整条 add 管线）：
// base 的段原文与顺序逐字节保留，fresh 独有的段追加在尾部，base 独有的段（用户自有）不丢。
func TestMergeSectionBlocksKeepsBaseAndAppendsFresh(t *testing.T) {
	base := []byte("# 头注释\nalpha:\n  v: 1 # 用户改过\n\n# 用户自有段\nmine:\n  flag: true\n")
	fresh := []byte("alpha:\n  v: 2\nbeta:\n  v: 9\n")
	merged, err := MergeSectionBlocks(base, fresh)
	require.NoError(t, err)
	assert.Equal(t, string(base)+"beta:\n  v: 9\n", string(merged), "base 为底 + fresh 独有段追加")

	// base == fresh（无手改）时逐字节等价：这是「二次 add 幂等」的段级依据。
	same, err := MergeSectionBlocks(fresh, fresh)
	require.NoError(t, err)
	assert.Equal(t, string(fresh), string(same))

	// 段解析失败必须报错（绝不静默丢内容）。
	_, err = MergeSectionBlocks([]byte("  orphan: 1\n"), fresh)
	require.ErrorContains(t, err, "section parser")
}

// TestMergeSectionBlocksInsertsSeparatorWhenBaseLacksTrailingNewline 是 Minor ② 的护栏：base 无尾
// 换行时接缝必须补一个换行 —— 否则 base 的末行段值与 fresh 的首个段键粘成一行，静默毁掉两段
// （`prefix: jimustorage:` 让 YAML 解析失败），或把下一段的注释粘进上一段的值里。
func TestMergeSectionBlocksInsertsSeparatorWhenBaseLacksTrailingNewline(t *testing.T) {
	// 键接缝：不补换行则 `prefix: jimustorage:` 直接让 YAML 解析失败。
	base := []byte("cache:\n  prefix: jimu")
	fresh := []byte("cache:\n  prefix: jimu\nstorage:\n  driver: local\n")
	merged, err := MergeSectionBlocks(base, fresh)
	require.NoError(t, err)
	assert.Equal(t, "cache:\n  prefix: jimu\nstorage:\n  driver: local\n", string(merged))

	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(string(merged))), "合并结果必须是合法 YAML")
	assert.Equal(t, "jimu", v.GetString("cache.prefix"))
	assert.Equal(t, "local", v.GetString("storage.driver"))

	// 注释接缝（审查者实测的形状）：不补换行时 YAML 仍能解析，但下一段的注释会被粘进上一段的值里。
	commentFresh := []byte("cache:\n  prefix: jimu\n# 文件存储配置\nstorage:\n  driver: local\n")
	merged2, err := MergeSectionBlocks([]byte("cache:\n  prefix: jimu"), commentFresh)
	require.NoError(t, err)
	v2 := viper.New()
	v2.SetConfigType("yaml")
	require.NoError(t, v2.ReadConfig(strings.NewReader(string(merged2))))
	assert.Equal(t, "jimu", v2.GetString("cache.prefix"), "注释不得粘进 cache.prefix 的值")
	assert.Equal(t, "local", v2.GetString("storage.driver"))

	// 无 fresh 独有段时 base 逐字节不变（不因补换行而多出字节）。
	same, err := MergeSectionBlocks(base, []byte("cache:\n  v: 2\n"))
	require.NoError(t, err)
	assert.Equal(t, string(base), string(same))
}

// TestDeclaredOrderNeverDuplicates 直接钉住声明集构造的**幂等不变量**：同一名字只出现一次，
// Ungated 集合与声明集同源。这是 Important ①（`add <Ungated> --force` 重复写入）的单元级证明。
func TestDeclaredOrderNeverDuplicates(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, d := range catalog.All() {
		inCatalog[d.Name] = true
	}
	cases := []struct {
		name     string
		existing []string
		add      string
		want     []string
		wantUng  []string
	}{
		{"已声明的 Ungated + --force（曾经重复）",
			[]string{"user", "access", "auth", "encryption", "notification"}, "encryption",
			[]string{"user", "access", "auth", "encryption", "notification"},
			[]string{"encryption", "notification"}},
		{"唯一的 Ungated + --force（--with=ws）",
			[]string{"ws"}, "ws", []string{"ws"}, []string{"ws"}},
		{"marker 手改出重复项：按首次出现去重",
			[]string{"ws", "ws", "apidocs"}, "ws", []string{"ws", "apidocs"}, []string{"ws", "apidocs"}},
		{"新增 catalog 能力：拓扑序插入",
			[]string{"user"}, "queue", []string{"user", "queue"}, nil},
		{"新增 Ungated 能力：追加在尾部",
			[]string{"user"}, "apidocs", []string{"user", "apidocs"}, []string{"apidocs"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			declared, ungated, err := declaredOrder(tc.existing, tc.add, inCatalog)
			require.NoError(t, err)
			assert.Equal(t, tc.want, declared)
			assert.Equal(t, tc.wantUng, ungated)
			for i, n := range declared {
				assert.NotContains(t, declared[i+1:], n, "声明集重复出现 %q：%v", n, declared)
			}
		})
	}
}

// TestAddCapabilityForceOnUngatedIsIdempotent 是 Important ① 的端到端护栏：Ungated（非 catalog）
// 能力用 --force 重建时声明集必须只出现一次。重复声明在生成期全绿（build/gofmt/5 条门禁都过），
// 只在进程启动 `assembly.validateAssembly` 时报 `capability "x" declared twice` —— 必须在这里挡住。
func TestAddCapabilityForceOnUngatedIsIdempotent(t *testing.T) {
	requireHeavyMatrix(t)
	root := frameworkRootForTest(t)
	cases := []struct {
		name string
		opts NewOptions
		cap  string
	}{
		{"minimal + encryption --force", NewOptions{Profile: "minimal"}, "encryption"},
		{"with ws + ws --force", NewOptions{With: "ws", Shape: "app"}, "ws"},
	}
	cache := newTestGoCache(t) // 两个用例共用一份专用缓存（cmd/server 只全量编译一次）
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := generateForTest(t, tc.opts)
			res, err := AddCapability(AddOptions{Dir: dir, Name: tc.cap, From: root, Force: true})
			require.NoError(t, err)
			assert.Empty(t, res.Changed, "重建同一集合必须无改动（幂等）")

			m, err := LoadMarker(dir)
			require.NoError(t, err)
			assert.Equal(t, 1, countIn(m.Capabilities, tc.cap), "marker 声明集重复：%v", m.Capabilities)
			assert.Equal(t, 1, countIn(catalogEntryNames(t, dir), tc.cap), "catalog entries 重复")
			assert.Equal(t, 1, assemblyDeclarationCount(t, dir, m.Shape, tc.cap), "assembly 重复声明")

			// 启动期装配校验（assembly.Run 第一步就是 validateAssembly）。env 已最小化（见
			// serverEnvForTest）→ prod 配置校验必然快速失败，不需要 DB；断言只要求进程走过
			// 装配校验进入「配置加载 / 连库」阶段，且不得出现装配错误或 panic。
			assertProjectBuilds(t, dir, cache)
			out := runServerForTest(t, dir, cache)
			assert.NotContains(t, out, "declared twice", "生成项目启动期装配校验失败：\n%s", out)
			assert.NotContains(t, out, "panic:", "启动期不得 panic：\n%s", out)
			assert.Regexp(t, `load config|database`, out, "装配校验必须通过并走到配置/连库阶段：\n%s", out)
		})
	}
}

// countIn 统计列表里某个名字的出现次数（Go 1.23 的 slices.Count 在本仓工具链上不可用）。
func countIn(list []string, name string) int {
	n := 0
	for _, s := range list {
		if s == name {
			n++
		}
	}
	return n
}

// assemblyDeclarationCount 统计生成版 internal/profiles/<shape>/assembly.go 里某个能力的声明条数。
func assemblyDeclarationCount(t *testing.T, dir, shape, name string) int {
	t.Helper()
	rel := filepath.ToSlash(filepath.Join("internal", "profiles", shape, "assembly.go"))
	return strings.Count(readProjectFile(t, dir, rel), name+"module.Descriptor")
}

// runServerForTest 在生成项目里跑一次 cmd/server（APP_ENV=prod 走快速失败的配置校验），返回合并输出。
func runServerForTest(t *testing.T, dir, cache string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/server")
	cmd.Dir = dir
	cmd.Env = serverEnvForTest(cache)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// serverEnvForTest 返回运行生成项目 cmd/server 的**最小显式 env**，绝不 `append(os.Environ(), …)`：
// `make` 经 `include .env` + `export` 把标准本地 checkout 的 .env（README 要求 `cp .env.example .env`）
// 整套注入子进程；APP_ENV/DB_*/JWT_SECRET/REDIS_* 齐备时 prod 配置校验会通过，服务器真的去连
// 127.0.0.1:3306 并重试（10×5s），断言随之变慢/失败 —— 这是正常本地开发态，必须与测试无关。
//
// 只透传 go 工具链自身的变量（自定义 GOPATH/GOMODCACHE/GOROOT/GOPROXY 的机器照样能跑），
// 应用配置键（APP_ENV 由本函数固定为 prod，其余 DB_*/JWT_SECRET/REDIS_* 一律不传）；
// GOFLAGS 由本函数固定为带 -trimpath（见 trimpathGoflags），不继承外部的 GOFLAGS。
func serverEnvForTest(cache string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GOWORK=off",
		"GOCACHE=" + cache,
		"GOFLAGS=" + trimpathGoflags(),
		"APP_ENV=prod",
	}
	for _, k := range []string{"GOPATH", "GOMODCACHE", "GOROOT", "GOTOOLCHAIN", "GOPROXY", "GOPRIVATE", "GONOSUMDB", "GOSUMDB"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}
