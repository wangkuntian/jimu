package db_test

import (
	"testing"

	"jimu/internal/platform/db"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/require"
)

// TestPostgresMigrationUpDown 对真实 PostgreSQL 跑 goose up + down，
// 验证 migrations/postgres/ 三个合并文件可完整应用与回滚。
// CI 通过 postgres service 提供；本地无 DB 时由 SkipUnlessPostgres 跳过。
func TestPostgresMigrationUpDown(t *testing.T) {
	tdb := testutil.SkipUnlessPostgres(t)
	defer tdb.Close()

	cfg := tdb.Config()

	// up：应建出核心表
	require.NoError(t, db.Migrate(cfg, "up"), "postgres up 应成功")
	for _, table := range []string{"users", "roles", "permissions", "user_roles", "role_permissions",
		"audit_logs", "outbox_events", "api_keys", "jobs", "job_history", "dead_letters",
		"import_jobs", "user_oauth_bindings", "scheduled_jobs"} {
		var n int
		require.NoError(t, tdb.Raw("SELECT count(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&n).Error,
			"查表 %s 失败", table)
		require.Equal(t, 1, n, "up 后应存在表 %s", table)
	}

	// reset：应回滚全部迁移，删除所有表
	require.NoError(t, db.Migrate(cfg, "reset"), "postgres reset 应成功")
	for _, table := range []string{"scheduled_jobs", "user_oauth_bindings", "import_jobs", "dead_letters",
		"job_history", "jobs", "api_keys", "outbox_events", "audit_logs", "role_permissions",
		"user_roles", "permissions", "roles", "users"} {
		var n int
		require.NoError(t, tdb.Raw("SELECT count(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&n).Error,
			"查表 %s 失败", table)
		require.Equal(t, 0, n, "reset 后应不存在表 %s", table)
	}
}
