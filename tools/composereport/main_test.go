package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"jimu/internal/contract"
	"jimu/internal/profiles/registry"
	"jimu/tools/internal/heavydeps"
	"jimu/tools/internal/profileoverlay"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModule 是 routeCount 的最小 Module 替身：按 paths 注册 GET 路由。
type fakeModule struct {
	name  string
	paths []string
}

func (m fakeModule) Name() string { return m.name }

func (m fakeModule) RegisterHTTP(r contract.Router) {
	for _, p := range m.paths {
		r.GET(p, func(*gin.Context) {})
	}
}

func (m fakeModule) RegisterJobs(contract.JobRegistry) {}

func (m fakeModule) RegisterEvents(contract.EventBus) {}

// TestTableCountUnionsOwnedTables 表数是各 Descriptor.Owns 的并集：重名只算一次。
func TestTableCountUnionsOwnedTables(t *testing.T) {
	got := tableCount([]contract.Descriptor{
		{Name: "a", Owns: []string{"users", "roles"}},
		{Name: "b", Owns: []string{"roles", "permissions"}},
		{Name: "c"},
	})
	assert.Equal(t, 3, got)
}

// TestMigrationCountWalksMysqlScripts 迁移数只数 embed FS 里 migrations/mysql 下的 .sql；
// postgres 与 mysql 同名同数，数一遍不重复计数。
func TestMigrationCountWalksMysqlScripts(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/mysql/001_a.sql":    &fstest.MapFile{},
		"migrations/mysql/002_b.sql":    &fstest.MapFile{},
		"migrations/mysql/README.md":    &fstest.MapFile{},
		"migrations/postgres/001_a.sql": &fstest.MapFile{},
		"migrations/postgres/002_b.sql": &fstest.MapFile{},
	}
	got, err := migrationCount([]contract.Descriptor{
		{Name: "a", Migrations: fsys},
		{Name: "b"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, got)
}

// TestRouteCountRegistersIntoBareEngine 路由数是模块在裸 gin.Engine 上注册出的条数之和；
// 没有路由的模块（纯迁移/端口能力）贡献 0。
func TestRouteCountRegistersIntoBareEngine(t *testing.T) {
	got := routeCount([]contract.Module{
		fakeModule{name: "a", paths: []string{"/api/v1/a", "/api/v1/a/:id"}},
		fakeModule{name: "b", paths: []string{"/api/v1/b"}},
		fakeModule{name: "no-routes"},
	})
	assert.Equal(t, 3, got)
}

// TestRenderReportPinsTheCommittedShape 报告是入库产物：用手写 Metrics 钉住表头、归一化列、
// 验收断言与「go.mod 逐形态相同」的结论，渲染逻辑变化会让本用例失败。
func TestRenderReportPinsTheCommittedShape(t *testing.T) {
	ms := []Metrics{
		{Profile: "full", BinaryBytes: 200_000_000, Routes: 100, Migrations: 20, Tables: 20, Files: 300, Lines: 30_000, HeavyDeps: []string{"aws-sdk-go-v2", "excelize"}, Capabilities: []string{"auth", "user"}},
		{Profile: "minimal", BinaryBytes: 100_000_000, Routes: 30, Migrations: 5, Tables: 5, Files: 150, Lines: 10_000, Capabilities: []string{"auth"}},
	}
	out := renderReport(ms, 64)

	assert.Contains(t, out, "| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 | 重型依赖 |")
	assert.Contains(t, out, "| `full` | 200.0 | 100.0% | 100 | 20 | 20 | 300 | 30000 | aws-sdk-go-v2, excelize |")
	assert.Contains(t, out, "| `minimal` | 100.0 | 50.0% | 30 | 5 | 5 | 150 | 10000 | - |")
	assert.Contains(t, out, "| 重型依赖 | 同一闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |")
	assert.Contains(t, out, "| 二进制 | `go build -overlay=<该形态> -o <tmp> ./cmd/server` 的产物大小 |")
	assert.Contains(t, out, "| 本仓 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./cmd/server` 在该形态 overlay 下的 import 闭包，只统计本模块（`jimu/...`）的非 `_test.go` 文件 |")
	assert.Contains(t, out, "形态由 `internal/profiles/active` 的**构建期 overlay** 决定")
	assert.Contains(t, out, "- 二进制：`minimal` 是 `full` 的 50.0%（要求 ≤ 85%）")
	assert.Contains(t, out, "- 路由数：`minimal` 30 < `full` 100")
	assert.Contains(t, out, "五个形态的 go.mod 直接依赖数**逐形态完全相同**（各 64 个）")
	assert.Contains(t, out, "| `minimal` | auth |")
}

// TestRenderReportWithoutMinimal 无 minimal 时不渲染验收段（只按数据渲染，不对形态名做隐藏假设）；
// 无 full 基准时同样跳过。
func TestRenderReportWithoutMinimal(t *testing.T) {
	out := renderReport([]Metrics{{Profile: "full", BinaryBytes: 2}}, 64)
	assert.NotContains(t, out, "要求 ≤ 85%")
	assert.Contains(t, out, "| `full` | 0.0 | 100.0% |")
}

// TestPercentAndMB 归一化列与 MB 呈现的边界：base 为 0 时不得除零，末位按一位小数四舍五入。
func TestPercentAndMB(t *testing.T) {
	assert.Equal(t, "—", percent(1, 0))
	assert.Equal(t, "50.0%", percent(1, 2))
	assert.Equal(t, "0.0", mb(0))
	assert.Equal(t, "1.5", mb(1_500_000))
}

// TestDirectDepsMeasuresTheModule 直接依赖是 module 级指标，与形态无关（层②边界）。
func TestDirectDepsMeasuresTheModule(t *testing.T) {
	n, err := directDeps(repoRoot(t))
	require.NoError(t, err)
	assert.Positive(t, n)
}

// TestMigrationCountRejectsMissingMysqlDir 缺少 migrations/mysql 时明确报错，不静默算 0。
func TestMigrationCountRejectsMissingMysqlDir(t *testing.T) {
	_, err := migrationCount([]contract.Descriptor{{Name: "a", Migrations: fstest.MapFS{}}})
	require.Error(t, err)
}

// TestCountLines 行数口径：换行符个数，末行无换行时补 1，空文件 0 行。
func TestCountLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.go")
	for _, tc := range []struct {
		content string
		want    int
	}{
		{"a\nb\n", 2},
		{"a\nb", 2},
		{"", 0},
	} {
		require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o600))
		got, err := countLines(path)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "content %q", tc.content)
	}
}

// TestOverlayForProfileMatchesTheSharedPackage 报告统计闭包用的内存 overlay 必须与共享包
// tools/internal/profileoverlay 的输出逐字节相同：报告若再长出模板副本，度量的就不再是
// 出货二进制。
func TestOverlayForProfileMatchesTheSharedPackage(t *testing.T) {
	root := t.TempDir()
	shared, err := profileoverlay.ReplaceMap(root, "minimal")
	require.NoError(t, err)
	got, err := overlayForProfile(root, "minimal")
	require.NoError(t, err)
	assert.Equal(t, shared, got)

	_, err = overlayForProfile(root, "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// TestWriteOverlayMatchesTheSharedPackage 报告构建二进制用的 overlay JSON 也由共享包按形态
// 隔离写出（`.overlay/<profile>/`）：路径、active.go 内容与 Replace 映射逐字节一致。
func TestWriteOverlayMatchesTheSharedPackage(t *testing.T) {
	root := t.TempDir()
	path, err := writeOverlay(root, "enterprise")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(profileoverlay.Dir(root, "enterprise"), "overlay.json"), path)

	src, err := profileoverlay.Source("enterprise")
	require.NoError(t, err)
	active, err := os.ReadFile(filepath.Join(profileoverlay.Dir(root, "enterprise"), "active.go"))
	require.NoError(t, err)
	assert.Equal(t, src, string(active))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var cfg struct{ Replace map[string]string }
	require.NoError(t, json.Unmarshal(content, &cfg))
	assert.Equal(t,
		filepath.Join(profileoverlay.Dir(root, "enterprise"), "active.go"),
		cfg.Replace[filepath.Join(root, "internal", "profiles", "active", "assembly.go")])

	_, err = writeOverlay(root, "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// TestMinimalCompiledSurfaceIsMateriallySmaller 是 Task 7 的验收断言：minimal 的二进制、
// 路由、表与本仓代码量必须显著低于 full。断言的是实测关系（相对比例），不写死任何数字，
// 因此内核膨胀或能力增减都不会让用例误报 —— 只会让真正的裁剪失效暴露出来。
func TestMinimalCompiledSurfaceIsMateriallySmaller(t *testing.T) {
	if testing.Short() {
		t.Skip("构建 5 个形态二进制，-short 下跳过")
	}
	ms, err := measureAll(repoRoot(t))
	require.NoError(t, err)
	require.Len(t, ms, len(registry.Names()))

	byName := make(map[string]Metrics, len(ms))
	for _, m := range ms {
		byName[m.Profile] = m
	}
	fullM, ok := byName["full"]
	require.True(t, ok)
	minM, ok := byName["minimal"]
	require.True(t, ok)

	require.Positive(t, fullM.BinaryBytes)
	require.Positive(t, fullM.Routes)
	require.Positive(t, fullM.Tables)
	require.Positive(t, fullM.Files)

	// 二进制至少小 15%（实测约 -30%）；路由、表、本仓文件数必须严格更小。
	assert.LessOrEqual(t, float64(minM.BinaryBytes), 0.85*float64(fullM.BinaryBytes),
		"minimal 二进制应至少比 full 小 15%%（full %d, minimal %d）", fullM.BinaryBytes, minM.BinaryBytes)
	assert.Less(t, minM.Routes, fullM.Routes, "minimal 路由数应少于 full")
	assert.Less(t, minM.Tables, fullM.Tables, "minimal 表数应少于 full")
	assert.Less(t, minM.Files, fullM.Files, "minimal 本仓 Go 文件数应少于 full")
	assert.Less(t, minM.Lines, fullM.Lines, "minimal 本仓代码行数应少于 full")

	// 驱动拆包后的重型依赖列：full 编进全部四类驱动，其余形态的编译面为零
	// （可插拔的实际效果；enterprise 已收敛为 local + csv）。
	assert.ElementsMatch(t, heavydeps.Names(), fullM.HeavyDeps, "full 应含全部四类重型依赖")
	for _, name := range []string{"minimal", "saas", "enterprise", "machine"} {
		m, ok := byName[name]
		require.True(t, ok, "报告缺少形态 %s", name)
		assert.Empty(t, m.HeavyDeps, "%s 不应把重型依赖编进编译面", name)
	}
}

// repoRoot 返回仓库根（本文件位于 tools/composereport/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
