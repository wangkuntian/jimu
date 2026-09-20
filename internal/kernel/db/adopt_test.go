package db_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/kernel/db"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/require"
)

// adoptVersionTables 返回夹具能力对应的版本表与业务表名（与 migration_integration_test.go 一致）
var (
	adoptVersionTables  = []string{"goose_db_version_user", "goose_db_version_auditsvc"}
	adoptBusinessTables = []string{"capmig_users", "capmig_audit_events"}
)

// seedLegacyGlobalVersionTable 模拟存量实例：创建全局版本表并直接 INSERT 版本行，
// 不执行任何迁移 SQL。versions 为要写入的版本号列表。
func seedLegacyGlobalVersionTable(t *testing.T, tdb *testutil.TestDB, versions []int) {
	t.Helper()
	require.NoError(t, tdb.Exec("CREATE TABLE IF NOT EXISTS goose_db_version (version_id BIGINT NOT NULL)").Error)
	for _, v := range versions {
		require.NoError(t, tdb.Exec("INSERT INTO goose_db_version (version_id) VALUES (?)", v).Error)
	}
}

// versionIDs 读取版本表全部 version_id（升序）
func versionIDs(t *testing.T, tdb *testutil.TestDB, table string) []int64 {
	t.Helper()
	var ids []int64
	require.NoError(t, tdb.Raw("SELECT version_id FROM "+table+" ORDER BY version_id").Scan(&ids).Error)
	return ids
}

// TestAdoptCapabilities_NoRerunAfterAdopt 是设计 §7/P1 的验收主线：
// 存量库（全局版本表记到 V）adopt 后，各能力版本表已有基线行，
// 再次 MigrateEnabled(up) 不重放任何已应用迁移（业务表不会被建出来）。
func TestAdoptCapabilities_NoRerunAfterAdopt(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	defer tdb.Close()

	caps := testCaps(t)
	cfg := tdb.Config()

	// 清理上一轮遗留，保证从零开始
	cleanupTables(t, tdb, append([]string{"goose_db_version"}, append(adoptVersionTables, adoptBusinessTables...)...))

	// 1. 旧世界：全局版本表记录 1..15（模拟存量实例已应用到 V=15，不执行任何 SQL）
	seedLegacyGlobalVersionTable(t, tdb, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})

	// 2. adopt：user 基线 [1,2]，auditsvc 基线 [1]
	baseline, err := db.AdoptCapabilities(cfg, caps)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, baseline["user"], "user 应基线 001/002")
	require.Equal(t, []int64{1}, baseline["auditsvc"], "auditsvc 应基线 001")
	// goose 建版本表时自动插入 0 基线行
	require.Equal(t, []int64{0, 1, 2}, versionIDs(t, tdb, "goose_db_version_user"))
	require.Equal(t, []int64{0, 1}, versionIDs(t, tdb, "goose_db_version_auditsvc"))

	// 3. 新世界：再次 up 不得执行任何已应用迁移
	require.NoError(t, db.MigrateEnabled(cfg, caps, "up"))

	// 4. 版本表行数不变（无新登记）
	require.Equal(t, []int64{0, 1, 2}, versionIDs(t, tdb, "goose_db_version_user"))
	require.Equal(t, []int64{0, 1}, versionIDs(t, tdb, "goose_db_version_auditsvc"))
	// 5. 业务表未被创建：证明迁移被跳过而非重放
	for _, table := range adoptBusinessTables {
		require.Equal(t, 0, tableCount(t, tdb, table), "表 %s 不应被创建（迁移不应重放）", table)
	}
	// 6. 全局版本表保留不动
	require.Len(t, versionIDs(t, tdb, "goose_db_version"), 15)

	// 全量清理
	cleanupTables(t, tdb, append([]string{"goose_db_version"}, append(adoptVersionTables, adoptBusinessTables...)...))
}

// TestAdoptCapabilities_FreshDBErrors 全新库（无全局版本表）应返回引导错误
func TestAdoptCapabilities_FreshDBErrors(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	defer tdb.Close()

	caps := testCaps(t)
	cfg := tdb.Config()

	// 确保全局版本表不存在（全新库）
	cleanupTables(t, tdb, append([]string{"goose_db_version"}, append(adoptVersionTables, adoptBusinessTables...)...))

	_, err := db.AdoptCapabilities(cfg, caps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "migrate up", "错误信息应引导先执行 migrate up")
}

// TestAdoptCapabilities_PartialAdopt 存量 V 小于部分迁移版本时只基线旧版本，
// 其余由后续 MigrateEnabled(up) 正常执行。使用 solo 夹具：001/002 互相独立
// （各自 CREATE 一张表），基线 001 后 up 只需真实执行 002，无表依赖问题。
func TestAdoptCapabilities_PartialAdopt(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	defer tdb.Close()

	root := soloCaps(t)
	caps := []contract.Descriptor{{Name: "solo", Migrations: root}}
	cfg := tdb.Config()

	versionTable := "goose_db_version_solo"
	businessTables := []string{"capmig_solo_items", "capmig_solo_tags"}
	cleanupTables(t, tdb, append([]string{"goose_db_version"}, businessTables...))
	cleanupTables(t, tdb, []string{versionTable})

	// 旧世界：存量只应用到 V=1
	seedLegacyGlobalVersionTable(t, tdb, []int{1})

	// adopt：只基线 001
	baseline, err := db.AdoptCapabilities(cfg, caps)
	require.NoError(t, err)
	require.Equal(t, []int64{1}, baseline["solo"])
	require.Equal(t, []int64{0, 1}, versionIDs(t, tdb, versionTable))

	// up：002 真实执行建表；001 已基线，其建表语句不执行
	require.NoError(t, db.MigrateEnabled(cfg, caps, "up"))
	require.Equal(t, []int64{0, 1, 2}, versionIDs(t, tdb, versionTable))
	require.Equal(t, 0, tableCount(t, tdb, "capmig_solo_items"), "001 已基线，建表语句不应执行")
	require.Equal(t, 1, tableCount(t, tdb, "capmig_solo_tags"), "002 应真实执行建表")

	// 全量清理
	cleanupTables(t, tdb, append([]string{"goose_db_version"}, businessTables...))
	cleanupTables(t, tdb, []string{versionTable})
}

// soloCaps 构造 solo 夹具能力（001/002 互相独立的迁移，供部分基线场景使用）
func soloCaps(t *testing.T) (fsys fs.FS) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	root := filepath.Join(filepath.Dir(thisFile), "testdata", "capmigs")
	return os.DirFS(filepath.Join(root, "solo"))
}
