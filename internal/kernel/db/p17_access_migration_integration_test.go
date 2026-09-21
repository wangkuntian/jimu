package db_test

import (
	"testing"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/kernel/db"

	"github.com/stretchr/testify/require"
)

// TestAccessMigrationsApply 验证合并后的 access 能力迁移集在真实库上可应用：
// 同一能力内版本号唯一（回归：role 001 与 permission 001 合并曾导致 goose
// "found duplicate migration version 1"），四张表建出，down/up 幂等。
func TestAccessMigrationsApply(t *testing.T) {
	tdb := newTempCapabilityDB(t, "jimu_p17_access").TestDB
	caps := catalog.All()
	cfg := tdb.Config()

	require.NoError(t, db.Migrate(cfg, caps, "up"), "access 迁移应可应用")
	for _, table := range []string{"roles", "user_roles", "permissions", "role_permissions"} {
		require.True(t, tdb.DB.Migrator().HasTable(table), "表 %s 应存在", table)
	}

	// down 一轮：各能力回滚最后一条（access 回滚 008，roles 的 version 列删除）
	require.NoError(t, db.Migrate(cfg, caps, "down"))
	// 再 up 幂等
	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.True(t, tdb.DB.Migrator().HasTable("roles"))
	require.True(t, tdb.DB.Migrator().HasTable("permissions"))
}
