package projectmetrics

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

// 本文件是本仓「编译面度量」原语的单测（P2.7 从 tools/composereport/main_test.go 原样搬来，
// 口径逐条不变）：生成器与报告工具共用同一份实现，因此这些断言只此一处。

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
	got := TableCount([]contract.Descriptor{
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
	got, err := MigrationCount([]contract.Descriptor{
		{Name: "a", Migrations: fsys},
		{Name: "b"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, got)
}

// TestMigrationCountRejectsMissingMysqlDir 缺少 migrations/mysql 时明确报错，不静默算 0。
func TestMigrationCountRejectsMissingMysqlDir(t *testing.T) {
	_, err := MigrationCount([]contract.Descriptor{{Name: "a", Migrations: fstest.MapFS{}}})
	require.Error(t, err)
}

// TestRouteCountRegistersIntoBareEngine 路由数是模块在裸 gin.Engine 上注册出的条数之和；
// 没有路由的模块（纯迁移/端口能力）贡献 0。
func TestRouteCountRegistersIntoBareEngine(t *testing.T) {
	got := RouteCount([]contract.Module{
		fakeModule{name: "a", paths: []string{"/api/v1/a", "/api/v1/a/:id"}},
		fakeModule{name: "b", paths: []string{"/api/v1/b"}},
		fakeModule{name: "no-routes"},
	})
	assert.Equal(t, 3, got)
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
		got, err := CountLines(path)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "content %q", tc.content)
	}
}

// TestDirectDepsMeasuresTheModule 直接依赖是 module 级指标，与形态无关（层②边界）；
// 主模块自身不计入。
func TestDirectDepsMeasuresTheModule(t *testing.T) {
	root := repoRoot(t)
	all, err := DirectDeps(root, "jimu")
	require.NoError(t, err)
	assert.Positive(t, all)

	// 排除的主模块名不匹配时（口径漂移）计数会多 1 —— 这条钉住「主模块不计入」。
	other, err := DirectDeps(root, "example.com/other")
	require.NoError(t, err)
	assert.Equal(t, all+1, other, "主模块名匹配时恰好少计一个")
}

// repoRoot 返回仓库根（本文件位于 tools/internal/projectmetrics/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
