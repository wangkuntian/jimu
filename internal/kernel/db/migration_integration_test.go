package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/kernel/db"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/require"
)

// TestMigrationIntegration 对真实 MySQL/Postgres 跑能力迁移运行器 MigrateEnabled：
// 两个测试夹具能力（testdata/capmigs 下）各用独立版本表 goose_db_version_<name>。
// 验证：迁移应用成功、版本表分能力独立、二次执行幂等。
// CI 通过 mariadb/postgres service 提供；本地无 DB 时由 SkipUnlessDB 自动跳过
// （DB_DRIVER=postgres 切换方言）。
func TestMigrationIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	defer tdb.Close()

	caps := testCaps(t)
	cfg := tdb.Config()
	versionTables := []string{"goose_db_version_user", "goose_db_version_auditsvc"}
	businessTables := []string{"capmig_users", "capmig_audit_events"}

	// 清理上一轮遗留，保证从零开始
	cleanupTables(t, tdb, append(versionTables, businessTables...))

	// 第一次 up：迁移应全部应用
	require.NoError(t, db.MigrateEnabled(cfg, caps, "up"), "首次 up 应成功")
	for _, table := range versionTables {
		require.Equal(t, 1, tableCount(t, tdb, table), "版本表 %s 应存在", table)
	}
	for _, table := range businessTables {
		require.Equal(t, 1, tableCount(t, tdb, table), "业务表 %s 应存在", table)
	}

	// 版本表记录数：user 3 行（goose 的 0 版基线 + 2 条迁移）、auditsvc 2 行（0 版基线 + 1 条迁移）
	require.Equal(t, int64(3), rowCount(t, tdb, "goose_db_version_user"), "user 版本表应有 3 条记录")
	require.Equal(t, int64(2), rowCount(t, tdb, "goose_db_version_auditsvc"), "auditsvc 版本表应有 2 条记录")

	// 二次 up：幂等，版本表记录数不变
	require.NoError(t, db.MigrateEnabled(cfg, caps, "up"), "二次 up 应幂等成功")
	require.Equal(t, int64(3), rowCount(t, tdb, "goose_db_version_user"))
	require.Equal(t, int64(2), rowCount(t, tdb, "goose_db_version_auditsvc"))

	// 全量清理（测试自洁，不污染共享测试库上的其他用例）
	cleanupTables(t, tdb, append(versionTables, businessTables...))
}

// testCaps 构造夹具能力清单：迁移文件在 testdata/capmigs 下，
// 与真实能力同构（根目录 migrations/{mysql,postgres}/）。
func testCaps(t *testing.T) []contract.Descriptor {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	root := filepath.Join(filepath.Dir(thisFile), "testdata", "capmigs")

	userFS := os.DirFS(filepath.Join(root, "user"))
	auditFS := os.DirFS(filepath.Join(root, "auditsvc"))
	return []contract.Descriptor{
		{Name: "user", Migrations: userFS},
		{Name: "auditsvc", Migrations: auditFS},
		{Name: "admin"}, // nil Migrations：运行器必须跳过
	}
}

// tableCount information_schema 中同名表数量（1 = 存在）
func tableCount(t *testing.T, tdb *testutil.TestDB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, tdb.Raw("SELECT count(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&n).Error)
	return n
}

// rowCount 表内记录数
func rowCount(t *testing.T, tdb *testutil.TestDB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, tdb.Raw("SELECT count(*) FROM "+table).Scan(&n).Error)
	return n
}

// cleanupTables 删除测试产生的表（不存在时报错可忽略）
func cleanupTables(t *testing.T, tdb *testutil.TestDB, tables []string) {
	t.Helper()
	for _, table := range tables {
		_ = tdb.Exec("DROP TABLE IF EXISTS " + table).Error
	}
}
