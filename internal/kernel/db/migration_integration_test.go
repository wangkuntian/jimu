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
	versionTables := []string{"goose_db_version_user", "goose_db_version_auditsvc",
		"goose_db_version_zsvc", "goose_db_version_advc"} // zsvc/advc: testCaps 增补的不等深夹具
	businessTables := []string{"capmig_users", "capmig_audit_events", "capmig_zitems", "capmig_zshared"}

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

	// 连续三轮 down 均应成功（回归：down 按反向能力序回滚，且全部回滚后
	// goose.ErrNoNextVersion 视为完成——旧实现正向序在第二轮报 Table doesn't exist）
	require.NoError(t, db.MigrateEnabled(cfg, caps, "down"), "第 1 轮 down 应成功")
	require.Equal(t, int64(2), rowCount(t, tdb, "goose_db_version_user"), "user 应回滚 002")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_auditsvc"), "auditsvc 应回滚 001")
	require.Equal(t, 0, tableCount(t, tdb, "capmig_audit_events"), "auditsvc 业务表应被删除")
	require.Equal(t, 1, tableCount(t, tdb, "capmig_users"), "user 业务表应保留（note 列已删）")
	require.Equal(t, 0, columnCount(t, tdb, "capmig_users", "note"), "user 002 的 Down 应已删除 note 列")

	require.NoError(t, db.MigrateEnabled(cfg, caps, "down"), "第 2 轮 down 应成功")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_user"), "user 应回滚 001")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_auditsvc"), "auditsvc 已空，行数不变")

	require.NoError(t, db.MigrateEnabled(cfg, caps, "down"), "第 3 轮 down（已全部回滚）应成功")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_user"), "仅剩 0 基线行")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_auditsvc"))
	require.Equal(t, 0, tableCount(t, tdb, "capmig_users"), "user 业务表应被删除")

	// 全量清理（测试自洁，不污染共享测试库上的其他用例）
	cleanupTables(t, tdb, append(versionTables, businessTables...))
}

// TestMigrationIntegration_UnequalDepths 回归 F2 第二问（不等深夹具）：
// 基础能力 zsvc（2 条迁移，002 建 capmig_zshared）在前、依赖能力 advc（1 条迁移，
// ALTER capmig_zshared）在后，传入序仿真实清单的拓扑序 [基础, 依赖]。各能力每轮
// down 只回滚一条迁移：
//   - 轮 1：advc 回滚 001（删 tag 列），zsvc 回滚 002——若序错（zsvc 后回滚），
//     zsvc 002 的 Down DROP TABLE capmig_zshared 会先于 advc 001 Down 执行而报错；
//   - 轮 2：advc 已空，zsvc 回滚 001；
//   - 轮 3：两者均空，纯哨兵轮。
//
// 名字陷阱：zsvc > advc 按名降序——旧 sort 实现（Name 降序）在此传入序下第 1 轮
// down 即失败（zsvc 002 Down 先删 capmig_zshared，advc 001 Down 撞
// "Table doesn't exist"）；正向序同样失败。只有清单反转序（advc 先、zsvc 后）通过。
func TestMigrationIntegration_UnequalDepths(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	defer tdb.Close()

	caps := testCaps(t)
	// 只取不等深夹具能力，保持真实清单式拓扑序：基础(zsvc)在前、依赖(advc)在后
	var pair []contract.Descriptor
	for _, c := range caps {
		if c.Name == "zsvc" || c.Name == "advc" {
			pair = append(pair, c)
		}
	}
	require.Equal(t, []string{"zsvc", "advc"}, []string{pair[0].Name, pair[1].Name},
		"夹具传入序应为 [基础 zsvc, 依赖 advc]，否则本测试失去钉住意义")

	cfg := tdb.Config()
	versionTables := []string{"goose_db_version_zsvc", "goose_db_version_advc"}
	businessTables := []string{"capmig_zitems", "capmig_zshared"}
	cleanupTables(t, tdb, append(versionTables, businessTables...))

	require.NoError(t, db.MigrateEnabled(cfg, pair, "up"), "up 应成功")
	require.Equal(t, 1, tableCount(t, tdb, "capmig_zitems"))
	require.Equal(t, 1, tableCount(t, tdb, "capmig_zshared"))
	require.Equal(t, 1, columnCount(t, tdb, "capmig_zshared", "tag"), "advc 001 的 tag 列应存在")
	require.Equal(t, int64(3), rowCount(t, tdb, "goose_db_version_zsvc"), "zsvc 版本表应有 3 行")
	require.Equal(t, int64(2), rowCount(t, tdb, "goose_db_version_advc"), "advc 版本表应有 2 行")

	// 三轮 down 全部成功（旧 sort 实现第 1 轮即报 "Table doesn't exist"）
	require.NoError(t, db.MigrateEnabled(cfg, pair, "down"), "第 1 轮 down 应成功：依赖先回滚")
	require.Equal(t, 0, columnCount(t, tdb, "capmig_zshared", "tag"), "advc 001 Down 应删 tag 列")
	require.Equal(t, 0, tableCount(t, tdb, "capmig_zshared"), "zsvc 002 Down 应删 capmig_zshared")
	require.Equal(t, 1, tableCount(t, tdb, "capmig_zitems"), "zsvc 001 尚未回滚，capmig_zitems 应保留")
	require.Equal(t, int64(2), rowCount(t, tdb, "goose_db_version_zsvc"), "zsvc 应回滚 002")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_advc"), "advc 应回滚 001，仅剩 0 基线")

	require.NoError(t, db.MigrateEnabled(cfg, pair, "down"), "第 2 轮 down 应成功")
	require.Equal(t, 0, tableCount(t, tdb, "capmig_zitems"), "zsvc 001 Down 应删 capmig_zitems")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_zsvc"), "zsvc 仅剩 0 基线行")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_advc"), "advc 已空，行数不变")

	require.NoError(t, db.MigrateEnabled(cfg, pair, "down"), "第 3 轮 down（已全部回滚）应成功")
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_zsvc"))
	require.Equal(t, int64(1), rowCount(t, tdb, "goose_db_version_advc"))

	cleanupTables(t, tdb, append(versionTables, businessTables...))
}

// testCaps 构造夹具能力清单：迁移文件在 testdata/capmigs 下，
// 与真实能力同构（根目录 migrations/{mysql,postgres}/）。
// user(2 条)/auditsvc(1 条) 为基础夹具；zsvc(2 条，002 建 capmig_zshared)/
// advc(1 条，ALTER capmig_zshared) 为不等深+跨能力依赖夹具（见 README）。
func testCaps(t *testing.T) []contract.Descriptor {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	root := filepath.Join(filepath.Dir(thisFile), "testdata", "capmigs")

	userFS := os.DirFS(filepath.Join(root, "user"))
	auditFS := os.DirFS(filepath.Join(root, "auditsvc"))
	advcFS := os.DirFS(filepath.Join(root, "advc"))
	zsvcFS := os.DirFS(filepath.Join(root, "zsvc"))
	return []contract.Descriptor{
		{Name: "user", Migrations: userFS},
		{Name: "auditsvc", Migrations: auditFS},
		{Name: "admin"}, // nil Migrations：运行器必须跳过
		// 不等深依赖对：基础在前、依赖在后（真实清单同为拓扑序）
		{Name: "zsvc", Migrations: zsvcFS},
		{Name: "advc", Migrations: advcFS},
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

// columnCount information_schema 中表内指定列的数量（0 = 列不存在）
func columnCount(t *testing.T, tdb *testutil.TestDB, table, column string) int {
	t.Helper()
	var n int
	require.NoError(t, tdb.Raw("SELECT count(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?", table, column).Scan(&n).Error)
	return n
}

// cleanupTables 删除测试产生的表（不存在时报错可忽略）
func cleanupTables(t *testing.T, tdb *testutil.TestDB, tables []string) {
	t.Helper()
	for _, table := range tables {
		_ = tdb.Exec("DROP TABLE IF EXISTS " + table).Error
	}
}
