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
// 构建开销用信号量限流（并发 2）+ 随测试删除的专用 GOCACHE 缓解；不跳过。
func TestGeneratedProjectBuildsAndVetsForEverySelection(t *testing.T) {
	if testing.Short() {
		t.Skip("25 次真实构建在 -short 下跳过")
	}
	names := allCapabilityNames(t)
	// 25 次真实构建会产生大量链接产物：用**随测试自动删除**的专用 GOCACHE（见 newTestGoCache），
	// 避免往共享缓存里堆 25 份构建结果（实测共享缓存可涨到 25G，曾把磁盘写满）。
	// 全部子用例共用一个 module 路径，让内核那 ~200 个相同文件在缓存里去重。
	cache := newTestGoCache(t)
	sem := make(chan struct{}, 2) // 受限并发 2：并发链接对磁盘与 CPU 压力都大
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sem <- struct{}{}
			defer func() { <-sem }()

			dir := filepath.Join(t.TempDir(), "proj")
			_, err := NewProject(NewOptions{
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
	dir := filepath.Join(t.TempDir(), "proj")
	res, err := NewProject(NewOptions{Dir: dir, With: "breach", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	// breach → auth（根包）→ user/infrastructure（子包）+ Requires user/access；
	// auth 还 import encryption/notification/outbox/queue 的根包 → 一并整目录复制；
	// tenant 仍是 schema 依赖的迁移携带。
	assert.ElementsMatch(t,
		[]string{"access", "auth", "breach", "encryption", "notification", "outbox", "queue", "tenant", "user"},
		res.Capabilities)
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
	if testing.Short() {
		t.Skip("go run 服务器在 -short 下跳过")
	}
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{Dir: dir, Profile: "machine", Module: "example.com/proj", NoTidy: true})
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

// knownAssetGaps 是 T6 落地前**允许**的非编译失败：这些测试用 `os.ReadFile` 读
// docs/openapi/**（生成器按计划「docs/openapi 仅当含 apidocs 时携带」），在非 apidocs 选择下
// 文件不存在。它们能编译、vet 也过，只是运行期缺资产 —— 属 T6 的已知缺口（见报告 Fix round 3）。
var knownAssetGaps = map[string]bool{
	"TestOpenAPIIncludesCRUDContract": true,
}

// TestGeneratedProjectTestTreeIsGreen minimal/queue/machine 三例的生成项目 `go test ./...`：
// 保留下来的测试必须全部通过，**唯一**允许的失败是 knownAssetGaps 里登记的 T6 资产缺口
// （新出现的任何失败都会让本测试红）。
func TestGeneratedProjectTestTreeIsGreen(t *testing.T) {
	if testing.Short() {
		t.Skip("生成项目测试在 -short 下跳过")
	}
	for _, tc := range []struct {
		name string
		opts NewOptions
	}{
		{"with queue", NewOptions{With: "queue"}},
		{"profile minimal", NewOptions{Profile: "minimal"}},
		{"profile machine", NewOptions{Profile: "machine"}},
	} {
		cache := newTestGoCache(t)
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			tc.opts.Dir = dir
			tc.opts.Module = "example.com/proj"
			tc.opts.NoTidy = true
			_, err := NewProject(tc.opts)
			require.NoError(t, err)
			out, _ := runGoInProjectOutput(t, dir, cache, "test", "./...", "-count=1") // 允许 knownAssetGaps 的非 0 退出码
			failed := failedTestNames(out)
			for _, name := range failed {
				assert.True(t, knownAssetGaps[name], "生成项目出现未登记的失败测试 %s:\n%s", name, out)
			}
			if len(failed) == 0 {
				assert.NotContains(t, out, "FAIL", "生成项目 go test 失败:\n%s", out)
			}
			assert.NotContains(t, out, "[build failed]", "测试树必须可编译:\n%s", out)
		})
	}
}

// TestGeneratedTestTreePruningKeepsPackagesWhole 钉住 Important 4 的裁剪口径：
// 被丢弃的测试文件按**目录整组**（同目录测试互相引用符号，逐文件删会留下 undefined）。
func TestGeneratedTestTreePruningKeepsPackagesWhole(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{Dir: dir, With: "queue", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	m := readMarkerForTest(t, dir)

	// e2e 测试依赖 auth/access/user 的 domain 等未全部满足 → 整目录测试都被丢弃。
	discarded := map[string]bool{}
	for _, rel := range m.DiscardedTests {
		discarded[rel] = true
	}
	assert.Contains(t, discarded, "internal/e2e/helpers_test.go")
	for _, rel := range m.DiscardedTests {
		assert.True(t, strings.HasSuffix(rel, "_test.go"), "只丢弃测试文件：%s", rel)
	}
	// 整组口径的推论：某目录只要被裁剪，该目录下的测试文件必须**全部**在丢弃清单里。
	byDir := map[string][]string{}
	for _, rel := range m.DiscardedTests {
		dirRel := filepath.ToSlash(filepath.Dir(rel))
		byDir[dirRel] = append(byDir[dirRel], rel)
	}
	for dirRel, dropped := range byDir {
		kept := 0
		for _, rel := range m.Files {
			if strings.HasSuffix(rel, "_test.go") && filepath.ToSlash(filepath.Dir(rel)) == dirRel {
				kept++
			}
		}
		assert.Zero(t, kept, "目录 %s 被裁剪时必须整组丢弃，仍有 %d 个测试文件保留", dirRel, kept)
		assert.NotEmpty(t, dropped)
	}
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
	_, err := NewProject(NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	m := readMarkerForTest(t, dir)
	assert.NotEmpty(t, m.Files)
	assert.NotContains(t, m.Files, markerFile)
	assert.Equal(t, m.Files, sortedCopy(m.Files), "marker.files 必须排序")
	for _, rel := range m.Files {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}
	assert.Empty(t, m.Assets, "T6 前 assets 为空数组")
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
	_, err := NewProject(NewOptions{Dir: dir, With: "user,access", Module: "example.com/proj", NoTidy: true})
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

// TestNewProjectRejectsReportFlag Minor 9：--report 在 T2 阶段明确报错（不是静默 no-op）。
func TestNewProjectRejectsReportFlag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", Report: true, NoTidy: true})
	require.ErrorContains(t, err, "--report")
	require.ErrorContains(t, err, "T8")
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "--report 报错时不得落盘")
}
