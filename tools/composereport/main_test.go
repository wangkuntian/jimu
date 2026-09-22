package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"jimu/internal/contract"

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
		{Profile: "full", BinaryBytes: 200_000_000, Routes: 100, Migrations: 20, Tables: 20, Files: 300, Lines: 30_000, Capabilities: []string{"auth", "user"}},
		{Profile: "minimal", BinaryBytes: 100_000_000, Routes: 30, Migrations: 5, Tables: 5, Files: 150, Lines: 10_000, Capabilities: []string{"auth"}},
	}
	out := renderReport(ms, 64)

	assert.Contains(t, out, "| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 |")
	assert.Contains(t, out, "| `full` | 200.0 | 100.0% | 100 | 20 | 20 | 300 | 30000 |")
	assert.Contains(t, out, "| `minimal` | 100.0 | 50.0% | 30 | 5 | 5 | 150 | 10000 |")
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

// TestMinimalCompiledSurfaceIsMateriallySmaller 是 Task 7 的验收断言：minimal 的二进制、
// 路由、表与本仓代码量必须显著低于 full。断言的是实测关系（相对比例），不写死任何数字，
// 因此内核膨胀或能力增减都不会让用例误报 —— 只会让真正的裁剪失效暴露出来。
func TestMinimalCompiledSurfaceIsMateriallySmaller(t *testing.T) {
	if testing.Short() {
		t.Skip("构建 5 个形态二进制，-short 下跳过")
	}
	ms, err := measureAll(repoRoot(t))
	require.NoError(t, err)
	require.Len(t, ms, len(profileNames))

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
}

// repoRoot 返回仓库根（本文件位于 tools/composereport/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
