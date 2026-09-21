package db_test

import (
	"testing"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/require"
)

// userMFAColumns 返回 users 表现有列名集合。
func userMFAColumns(t *testing.T, tdb *testutil.TestDB) map[string]bool {
	t.Helper()
	cols, err := tdb.DB.Migrator().ColumnTypes("users")
	require.NoError(t, err)
	out := make(map[string]bool, len(cols))
	for _, c := range cols {
		out[c.Name()] = true
	}
	return out
}

// TestUserMFAMigrationShape 验证 mfa 能力迁移 016：up 后 users 不再有 totp_* 列、
// user_mfa 表存在；down 后列恢复、表删除；再次 up 幂等。
// 无真实 DB 时跳过（CI 由 mariadb/postgres service 提供，DB_DRIVER 切方言）。
func TestUserMFAMigrationShape(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	cfg := tdb.Config()
	caps := catalog.All()

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.True(t, tdb.DB.Migrator().HasTable("user_mfa"), "016 应创建 user_mfa")
	cols := userMFAColumns(t, tdb)
	require.False(t, cols["totp_secret"], "016 后 users 不应再有 totp_secret")
	require.False(t, cols["totp_enabled"], "016 后 users 不应再有 totp_enabled")

	require.NoError(t, db.Migrate(cfg, caps, "down"))
	require.False(t, tdb.DB.Migrator().HasTable("user_mfa"), "016 Down 应删除 user_mfa")
	cols = userMFAColumns(t, tdb)
	require.True(t, cols["totp_secret"], "016 Down 应恢复 users.totp_secret")
	require.True(t, cols["totp_enabled"], "016 Down 应恢复 users.totp_enabled")

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.True(t, tdb.DB.Migrator().HasTable("user_mfa"))
	require.False(t, userMFAColumns(t, tdb)["totp_secret"], "再次 up 应幂等")
}

// TestUserMFATOTPDataMigration 验证存量 TOTP 密文在 016 中原样搬迁（不重新加解密），
// 且 Down 能把数据搬回 users；未启用 TOTP 的用户不产生 user_mfa 行。
func TestUserMFATOTPDataMigration(t *testing.T) {
	tdb := testutil.SkipUnlessDB(t)
	cfg := tdb.Config()
	caps := catalog.All()

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.NoError(t, db.Migrate(cfg, caps, "down")) // 回到 015：users 带 totp_*

	// 存量启用 TOTP 用户（totp_secret 直接写"密文"字符串，模拟既有 AES-GCM 值）；
	// PG 下 totp_enabled 是 SMALLINT，统一用 1/0 写入。
	require.NoError(t, tdb.DB.Exec(
		"INSERT INTO users (id, username, password, status, tenant_id, totp_secret, totp_enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,?,NOW(),NOW())",
		uint64(9001), "p16_mfa_user", "hash", 1, tenant.DefaultTenantID, "enc:v1:CIPHERTEXT", 1).Error)
	// 未启用用户：不应搬迁
	require.NoError(t, tdb.DB.Exec(
		"INSERT INTO users (id, username, password, status, tenant_id, totp_enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,NOW(),NOW())",
		uint64(9002), "p16_mfa_user2", "hash", 1, tenant.DefaultTenantID, 0).Error)

	require.NoError(t, db.Migrate(cfg, caps, "up")) // 执行 016 搬迁

	var secret string
	var enabled bool
	require.NoError(t, tdb.DB.Raw("SELECT totp_secret, totp_enabled FROM user_mfa WHERE user_id = ?", uint64(9001)).Row().Scan(&secret, &enabled))
	require.Equal(t, "enc:v1:CIPHERTEXT", secret, "密文应原样搬迁")
	require.True(t, enabled)

	var count int64
	require.NoError(t, tdb.DB.Raw("SELECT COUNT(*) FROM user_mfa WHERE user_id = ?", uint64(9002)).Scan(&count).Error)
	require.Zero(t, count, "未启用 TOTP 的用户不应搬迁")

	require.NoError(t, db.Migrate(cfg, caps, "down"))
	require.NoError(t, tdb.DB.Raw("SELECT totp_secret, totp_enabled FROM users WHERE id = ?", uint64(9001)).Row().Scan(&secret, &enabled))
	require.Equal(t, "enc:v1:CIPHERTEXT", secret, "Down 应把密文搬回 users")
	require.True(t, enabled)

	require.NoError(t, tdb.DB.Exec("DELETE FROM users WHERE id IN (?, ?)", uint64(9001), uint64(9002)).Error)
}
