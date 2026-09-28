package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allCapabilityNames 枚举框架的**全部**能力名（catalog 18 ∪ Ungated 7 = 25），顺序确定。
// 从 capabilityDescriptors 派生而不是写死清单：新增/删除能力时本网自动跟随；总数变化会 fail，
// 提醒把新能力纳入这条系统网。
func allCapabilityNames(t *testing.T) []string {
	t.Helper()
	descs, _, err := capabilityDescriptors(FrameworkRoot())
	require.NoError(t, err)
	names := make([]string, 0, len(descs))
	for _, d := range descs {
		names = append(names, d.Name)
	}
	require.Len(t, names, 25, "框架能力总数应为 catalog 18 ∪ Ungated 7 = 25；变化时请确认本网覆盖新能力")
	return names
}

// TestGeneratedProjectBuildsAndVetsForEverySelection 是 C1 的**系统性回归网**：
// 对全部 25 个能力逐个做单能力选择，生成项目必须 `go build ./...` 与 `go vet ./...` 全绿。
// `--with=breach` 曾被漏掉（breach → auth → user/infrastructure 的子包依赖未被映射回属主能力）；
// 7 个 Ungated 能力（apidocs/storage/notification/retention/ws/grpc/encryption）也在这条网里。
//
// 构建开销用信号量限流（并发 2）+ newTestGoCache 缓解；整条矩阵只在重型门控下跑
// （requireHeavyMatrix：25 次真实构建在 CI 冷缓存上会把默认 Test/Race 拖到 30 分钟以上）。
func TestGeneratedProjectBuildsAndVetsForEverySelection(t *testing.T) {
	requireHeavyMatrix(t)
	names := allCapabilityNames(t)
	// 25 次真实构建会产生大量链接产物：用 newTestGoCache（默认随测试自动删除；CI 的 Scaffold
	// Matrix job 经 JIMU_TEST_GOCACHE 复用一份跨运行的缓存），避免往共享缓存里堆构建结果
	// （实测共享缓存可涨到 25G，曾把磁盘写满）。
	// 全部子用例共用一个 module 路径，让内核那 ~200 个相同文件在缓存里去重。
	cache := newTestGoCache(t)
	sem := make(chan struct{}, 2) // 受限并发 2：并发链接对磁盘与 CPU 压力都大
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sem <- struct{}{}
			defer func() { <-sem }()

			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{
				Dir: dir, With: name, Module: "example.com/proj", NoTidy: true,
			})
			require.NoError(t, err)
			assertProjectBuilds(t, dir, cache)
			assertProjectVets(t, dir, cache)
		})
	}
}

// TestSelectedCapabilityIsAssembledWithItsWholeCompileClosure 用 breach 钉住 C1 的具体形态：
// breach import auth（根包），auth import user/infrastructure（**子包**）且 Requires user/access
// —— 三者都必须整目录复制，且 tenant 仍只作携带。
func TestSelectedCapabilityIsAssembledWithItsWholeCompileClosure(t *testing.T) {
	requireHeavyMatrix(t)
	dir := filepath.Join(t.TempDir(), "proj")
	res, err := newProjectForTest(t, NewOptions{Dir: dir, With: "breach", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	// breach → auth（根包）→ user/infrastructure（子包）+ Requires user/access；
	// auth 还 import encryption/notification/outbox/queue 的根包 → 一并整目录复制；
	// tenant 仍是 schema 依赖的迁移携带。
	assert.ElementsMatch(t,
		[]string{"access", "auth", "breach", "encryption", "notification", "outbox", "queue", "tenant", "user"},
		res.CopySet)
	// breach 的 auth 是**软依赖**（SoftRequires）：装配集只有 breach，auth 只是被它生产 import 到
	// 编译闭包里；复制集才是那 9 个（`Capabilities` vs `CopySet` 的差别由此可见）。
	assert.Equal(t, []string{"breach"}, res.Capabilities, "Capabilities 是装配集（Declared）")
	for _, p := range []string{
		"internal/capabilities/breach/wire.go",
		"internal/capabilities/auth/wire.go",
		"internal/capabilities/auth/application",    // auth 自己的子包
		"internal/capabilities/user/infrastructure", // auth import 的 user 子包（C1 的关键）
		"internal/capabilities/user/wire.go",
		"internal/capabilities/access/wire.go",
	} {
		_, statErr := os.Stat(filepath.Join(dir, filepath.FromSlash(p)))
		assert.NoError(t, statErr, "缺少编译闭包成员 %s", p)
	}
	assertProjectBuilds(t, dir, newTestGoCache(t))
}

// TestGeneratedServerSurvivesProtobufDescriptor 是 C2 的端到端验收：`--profile=machine`
// 生成项目里 `go run ./cmd/server` **不得**出现 protobuf 描述符解析 panic。
// 连库失败是预期（无 MySQL），只要不是 filedesc/slice bounds 崩溃即可。
func TestGeneratedServerSurvivesProtobufDescriptor(t *testing.T) {
	requireHeavyMatrix(t)
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "machine", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)

	// 先确认生成树里的 raw descriptor 逐字节完好（不是靠运行期侥幸）。
	pb := filepath.Join(dir, "internal/capabilities/grpc/userinfopb/userinfo.pb.go")
	content, err := os.ReadFile(pb)
	require.NoError(t, err, "machine 形态应含 grpc 与它的 .pb.go")
	assert.Contains(t, string(content), `\x1cproto/jimu/v1/userinfo.proto`)
	assert.NotContains(t, string(content), "proto/example.com")

	cache := newTestGoCache(t)
	got, _ := runGoInProjectOutput(t, dir, cache, "run", "./cmd/server") // 连库失败会让退出码非 0，属预期
	assert.NotContains(t, got, "filedesc")
	assert.NotContains(t, got, "slice bounds out of range")
	assert.NotContains(t, got, "panic:")
	assert.Contains(t, got, "database", "应当走到连库阶段（说明启动初始化没有崩）")
}

// failedTestNames 从 `go test` 输出里提取 `--- FAIL: <name>` 的测试名。
func failedTestNames(output string) []string {
	var names []string
	for _, line := range strings.Split(output, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "--- FAIL: ")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(rest, " ")
		names = append(names, name)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// TestGeneratedProjectTestTreeIsGreen minimal/queue/machine 三例的生成项目 `go test ./...`：
// 保留下来的测试必须**全部通过**。
//
// 曾经的唯一豁免（`TestOpenAPIIncludesCRUDContract` 读 docs/openapi/swagger.json）已按**资产依赖**
// 裁剪掉：引用未复制资产的测试文件（连同其所在目录的整组测试）不进生成树，故这里不再有任何
// 「已登记的失败」白名单 —— 新出现的任何失败都让本测试红。
func TestGeneratedProjectTestTreeIsGreen(t *testing.T) {
	requireHeavyMatrix(t)
	// 覆盖 5 个 profile + 单能力/小集合的「裁剪得最狠」选区：保留的测试必须真的能跑（不只是能编译）。
	// 全部用例共用一个专用 GOCACHE（依赖只编译一次），仍不碰共享缓存（见 projectbuild_test.go）。
	cache := newTestGoCache(t)
	for _, tc := range []struct {
		name string
		opts NewOptions
	}{
		{"profile full", NewOptions{Profile: "full"}},
		{"profile minimal", NewOptions{Profile: "minimal"}},
		{"profile saas", NewOptions{Profile: "saas"}},
		{"profile enterprise", NewOptions{Profile: "enterprise"}},
		{"profile machine", NewOptions{Profile: "machine"}},
		{"with queue", NewOptions{With: "queue"}},
		{"with storage", NewOptions{With: "storage"}},
		{"with apidocs", NewOptions{With: "apidocs"}},
		{"with retention", NewOptions{With: "retention"}},
		{"with grpc", NewOptions{With: "grpc"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			tc.opts.Dir = dir
			tc.opts.Module = "example.com/proj"
			tc.opts.NoTidy = true
			_, err := newProjectForTest(t, tc.opts)
			require.NoError(t, err)
			out, err := runGoInProjectOutput(t, dir, cache, "test", "./...", "-count=1")
			require.NoError(t, err, "生成项目 go test 必须全绿:\n%s", out)
			assert.NotContains(t, out, "[build failed]", "测试树必须可编译:\n%s", out)
		})
	}
}

// TestGeneratedTestTreePruningKeepsSatisfiableTestFiles 钉住裁剪口径的**逐文件**一半：
// 目录里有文件被裁，不代表整个目录都要裁 —— 逐文件 import/资产可满足性下合法的测试必须保留
// （内部/审查者重放：minimal 的 internal/app、internal/assembly、outbox、internal/contract 都有
// 可保留文件；此前按目录整组丢弃会多删 15 个文件/65 个 Test*）。
func TestGeneratedTestTreePruningKeepsSatisfiableTestFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	m := readMarkerForTest(t, dir)
	discarded := map[string]bool{}
	for _, rel := range m.DiscardedTests {
		discarded[rel] = true
	}
	for _, rel := range []string{
		"internal/app/application_test.go",            // 逐文件可满足
		"internal/app/bootstrap_http_test.go",         //
		"internal/assembly/assembly_test.go",          //
		"internal/capabilities/outbox/outbox_test.go", //
		"internal/contract/capability_test.go",        // 资产依赖只裁同目录的 openapi_test.go
	} {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(rel)), "%s 应当保留", rel)
		assert.False(t, discarded[rel], "%s 不应被裁", rel)
	}
	assert.True(t, discarded["internal/contract/openapi_test.go"], "读 docs/openapi 的测试必须裁掉")
	// 组成依赖：读生成期派生的组成清单（catalog 是专属子集）的测试不可移植 —— 实测 `--with=queue`
	// 时 seed_test.go 的 TestRunSeed_* 会因全量 permissions 期望落空而红，故按逐文件裁掉。
	for _, rel := range []string{
		"internal/app/seed_test.go",
		"internal/kernel/db/p17_access_migration_integration_test.go",
		"internal/kernel/db/user_mfa_migration_integration_test.go",
	} {
		assert.True(t, discarded[rel], "组成依赖的测试应被裁掉：%s", rel)
	}
	for _, rel := range m.DiscardedTests {
		assert.True(t, strings.HasSuffix(rel, "_test.go"), "只丢弃测试文件：%s", rel)
	}
}

// TestPruneUnsatisfiableTestsDropsCompositionDependentTests 组成依赖单独一条：import 解析得到
// （生成树里就有 `internal/capabilities/catalog`），但测试把「框架全量组成」当期望值 —— 生成项目的
// catalog 只是本项目子集，这类测试必须逐文件丢掉，否则在裁剪项目里运行期红。
func TestPruneUnsatisfiableTestsDropsCompositionDependentTests(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "internal/capabilities/catalog/catalog.go", "package catalog\n")
	writeFileForTest(t, dir, "internal/app/app.go", "package app\n")
	writeFileForTest(t, dir, "internal/app/seed_test.go", `package app

import (
	"testing"

	"example.com/proj/internal/capabilities/catalog"
)

func TestSeed(t *testing.T) { _ = catalog.All() }
`)
	writeFileForTest(t, dir, "internal/app/plain_test.go", `package app

import "testing"

func TestPlain(t *testing.T) {}
`)

	discarded, err := pruneUnsatisfiableTests(dir, "example.com/proj", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"internal/app/seed_test.go"}, discarded)
	assert.NoFileExists(t, filepath.Join(dir, "internal/app/seed_test.go"))
	assert.FileExists(t, filepath.Join(dir, "internal/app/plain_test.go"))
}

// TestGeneratedTestTreePruningFallsBackToWholeTestPackage 钉住裁剪口径的**包回退**一半：
// 同一测试包里若保留文件引用了被裁文件的顶层符号，逐文件保留会留下 `undefined: xxx` ——
// internal/e2e 正是这个形态（admin_routes_parity_test.go 引用被裁文件里的 newTestAppWithDB），
// 因此该测试包必须整组丢弃。
func TestGeneratedTestTreePruningFallsBackToWholeTestPackage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	m := readMarkerForTest(t, dir)
	discarded := map[string]bool{}
	for _, rel := range m.DiscardedTests {
		discarded[rel] = true
	}
	// e2e 的测试包被整组丢弃：目录里不得残留任何 _test.go，且每个原文件都在丢弃清单里。
	entries, rerr := os.ReadDir(filepath.Join(dir, "internal", "e2e"))
	require.NoError(t, rerr)
	for _, e := range entries {
		assert.False(t, strings.HasSuffix(e.Name(), "_test.go"), "e2e 测试包应整组丢弃，残留 %s", e.Name())
	}
	for _, rel := range []string{
		"internal/e2e/helpers_test.go",
		"internal/e2e/api_contract_test.go",
		"internal/e2e/admin_routes_parity_test.go",
	} {
		assert.True(t, discarded[rel], "%s 应随测试包整组丢弃", rel)
	}
}

// TestPruneUnsatisfiableTestsPerFileWithPackageFallback 用**人造树**把新口径的两条分支钉成单元
// 级断言（不依赖任何具体能力选择）：
//
//	a/：keep_test.go 引用被裁文件 use_test.go 的顶层符号 helperX → 整个测试包回退丢弃；
//	b/：drop_test.go 被裁、ok_test.go 不引用它 → 只丢 drop_test.go；
//	c/：`package c`（被裁）与 `package c_test`（保留）是两个测试包，跨包同名不得触发回退。
func TestPruneUnsatisfiableTestsPerFileWithPackageFallback(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "a/a.go", "package a\n")
	writeFileForTest(t, dir, "a/use_test.go", `package a

import _ "example.com/proj/internal/capabilities/ghost"

func helperX() {}
`)
	writeFileForTest(t, dir, "a/keep_test.go", `package a

import "testing"

func TestKeep(t *testing.T) { helperX() }
`)
	writeFileForTest(t, dir, "b/b.go", "package b\n")
	writeFileForTest(t, dir, "b/drop_test.go", `package b

import _ "example.com/proj/internal/capabilities/ghost"

func helperY() {}
`)
	writeFileForTest(t, dir, "b/ok_test.go", `package b

import "testing"

func TestOK(t *testing.T) {}
`)
	writeFileForTest(t, dir, "c/c.go", "package c\n")
	writeFileForTest(t, dir, "c/drop_test.go", `package c

import _ "example.com/proj/internal/capabilities/ghost"

func helperZ() {}
`)
	writeFileForTest(t, dir, "c/keep_test.go", `package c_test

import "testing"

func TestKeep(t *testing.T) { helperZ() }
`)

	discarded, err := pruneUnsatisfiableTests(dir, "example.com/proj", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"a/keep_test.go", // 包回退：引用了被裁文件里的 helperX
		"a/use_test.go",
		"b/drop_test.go", // 逐文件：ok_test.go 不引用它，保留
		"c/drop_test.go", // 包回退不跨测试包（`package c` vs `package c_test`）
	}, discarded)
	assert.NoFileExists(t, filepath.Join(dir, "a", "keep_test.go"))
	assert.NoFileExists(t, filepath.Join(dir, "a", "use_test.go"))
	assert.FileExists(t, filepath.Join(dir, "b", "ok_test.go"))
	assert.FileExists(t, filepath.Join(dir, "c", "keep_test.go"))
}

// sortedCopy 返回排序后的副本（校验 marker.files 已排序）。
func sortedCopy(in []string) []string {
	out := append([]string{}, in...)
	slices.Sort(out)
	return out
}

// TestMarkerRecordsFilesAndDiscardedTests Minor 10：marker 的 files/discardedTests 字段。
func TestMarkerRecordsFilesAndDiscardedTests(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	m := readMarkerForTest(t, dir)
	assert.NotEmpty(t, m.Files)
	assert.NotContains(t, m.Files, markerFile)
	assert.Equal(t, m.Files, sortedCopy(m.Files), "marker.files 必须排序")
	for _, rel := range m.Files {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}
	// T6：assets 是本次复制的资产路径（AssetsFor 的前缀口径）——minimal = 全部内核资产组路径，
	// 不含 docs/openapi（docs/openapi 只在含 apidocs 的选择里）。
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	wantAssets, err := AssetsFor(set)
	require.NoError(t, err)
	assert.Equal(t, wantAssets, m.Assets, "marker.assets 必须与 AssetsFor 同源")
	for _, rel := range m.Assets {
		_, serr := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		assert.NoError(t, serr, "资产路径 %s 必须真的落地", rel)
	}
	assert.NotContains(t, m.Assets, "docs/openapi")
}

// TestParseCapabilitySetAcceptsUngatedCapabilities Important 3：`--with` 用框架全量集合解析，
// 7 个 Ungated 能力同样可单独选中，并保持 Ungated 语义与 S4 默认驱动（storage→local）。
func TestParseCapabilitySetAcceptsUngatedCapabilities(t *testing.T) {
	for _, tc := range []struct {
		with        string
		wantUngated []string
		wantDrivers map[string][]string
	}{
		{"storage", []string{"storage"}, map[string][]string{"storage": {"local"}}},
		{"encryption", []string{"encryption"}, nil},
		{"apidocs,storage", []string{"apidocs", "storage"}, map[string][]string{"storage": {"local"}}},
	} {
		t.Run(tc.with, func(t *testing.T) {
			set, err := ParseCapabilitySet("", tc.with, "app")
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantUngated, set.Ungated)
			for name, want := range tc.wantDrivers {
				assert.Equal(t, want, set.Drivers[name])
			}
			assert.Contains(t, set.Declared, tc.wantUngated[0])
		})
	}
	// 未知名字仍然 fail-closed。
	_, err := ParseCapabilitySet("", "ghost", "app")
	require.ErrorContains(t, err, `unknown capability "ghost"`)
}

// TestGeneratedCatalogKnownCoversEveryCapability Minor 6：生成版 catalog 的 `known` 必须是
// 框架全量能力名（catalog 18 ∪ Ungated 7 = 25），语义是「软依赖指向缺席能力 = 降级」。
func TestGeneratedCatalogKnownCoversEveryCapability(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	// user/access 会带出 schema 依赖 tenant，正好覆盖 Minor 7。
	_, err := newProjectForTest(t, NewOptions{Dir: dir, With: "user,access", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "internal/capabilities/catalog/catalog.go"))
	require.NoError(t, err)
	for _, name := range allCapabilityNames(t) {
		assert.Contains(t, string(content), `"`+name+`",`, "known 缺少能力 %s", name)
	}
	// Minor 7：迁移携带的 tenant 必须进 entries（否则 MigrationSet 取不到，
	// `jimu migrate` 会漏 tenant 的建表/加列）。
	assert.Contains(t, string(content), "tenantmodule.Descriptor,")
	migration, err := os.ReadFile(filepath.Join(dir, "internal/capabilities/catalog/migration.go"))
	require.NoError(t, err)
	assert.Contains(t, string(migration), `"user"`)
	// T3 起渲染与框架蓝本同构：契约字面量写作 {"tenant"}（不再是 T2 占位的 []string{"tenant"}），
	// 且 migration.go 里**没有**第二条清单（migrationExtras 形态已按裁定修订第 1 条删除）。
	assert.Contains(t, string(migration), `{"tenant"}`)
	assert.NotContains(t, string(migration), "migrationExtras")
}

// heavyMatrixEnv / requireHeavyMatrix / newTestGoCache 见 projectbuild_test.go：本文件的重型用例
// 一律经 requireHeavyMatrix 门控，默认路径只跑生成/裁剪/解析这类便宜断言。
