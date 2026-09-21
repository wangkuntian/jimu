package db_test

import (
	"database/sql"
	"fmt"
	"testing"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/config"
	"jimu/internal/kernel/db"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/require"
)

// tempCapabilityDB 在共享测试库之外开一个**独立临时数据库**，跑完即删。
// 016 会 DROP users.totp_* 列，不能动共享 jimu_test（其他包同时跑 Migrate()，
// 共享库被回滚到旧结构会让它们撞 duplicate column）；这里完全隔离。
type tempCapabilityDB struct {
	*testutil.TestDB
	name  string
	admin *sql.DB // server 级连接，用于 CREATE/DROP DATABASE
}

func newTempCapabilityDB(t *testing.T, name string) *tempCapabilityDB {
	t.Helper()
	base := testutil.SkipUnlessDB(t).Config()

	adminCfg := base
	adminCfg.Database = ""
	admin, err := sql.Open(tempDBDriver(base.Driver), tempDBDSN(adminCfg))
	require.NoError(t, err)
	_, err = admin.Exec("DROP DATABASE IF EXISTS " + name)
	require.NoError(t, err)
	_, err = admin.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)

	cfg := base
	cfg.Database = name
	tdb, err := testutil.NewTestDB(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tdb.Close()
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name)
		_ = admin.Close()
	})
	return &tempCapabilityDB{TestDB: tdb, name: name, admin: admin}
}

func tempDBDriver(driver string) string {
	if driver == "postgres" || driver == "postgresql" {
		return "pgx"
	}
	return "mysql"
}

func tempDBDSN(cfg config.DBConfig) string {
	if cfg.Driver == "postgres" || cfg.Driver == "postgresql" {
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)
}

// tempDBUserColumns users 表现有列名集合。
func tempDBUserColumns(t *testing.T, tdb *testutil.TestDB) map[string]bool {
	t.Helper()
	cols, err := tdb.DB.Migrator().ColumnTypes("users")
	require.NoError(t, err)
	out := make(map[string]bool, len(cols))
	for _, c := range cols {
		out[c.Name()] = true
	}
	return out
}

// TestUserMFAMigrationShape 验证 mfa 迁移 016：up 后 users 不再有 totp_* 列、
// user_mfa 表存在；down 后列恢复、表删除；再次 up 幂等。
func TestUserMFAMigrationShape(t *testing.T) {
	tdb := newTempCapabilityDB(t, "jimu_p16_shape").TestDB
	caps := catalog.All()
	cfg := tdb.Config()

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.True(t, tdb.DB.Migrator().HasTable("user_mfa"), "016 应创建 user_mfa")
	cols := tempDBUserColumns(t, tdb)
	require.False(t, cols["totp_secret"], "016 后 users 不应再有 totp_secret")
	require.False(t, cols["totp_enabled"], "016 后 users 不应再有 totp_enabled")

	require.NoError(t, db.Migrate(cfg, caps, "down"))
	require.False(t, tdb.DB.Migrator().HasTable("user_mfa"), "016 Down 应删除 user_mfa")
	cols = tempDBUserColumns(t, tdb)
	require.True(t, cols["totp_secret"], "016 Down 应恢复 users.totp_secret")
	require.True(t, cols["totp_enabled"], "016 Down 应恢复 users.totp_enabled")

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.True(t, tdb.DB.Migrator().HasTable("user_mfa"))
	require.False(t, tempDBUserColumns(t, tdb)["totp_secret"], "再次 up 应幂等")
}

// TestUserMFATOTPDataMigration 验证存量 TOTP 密文在 016 中原样搬迁（不重新加解密），
// 且 Down 能把数据搬回 users；未启用 TOTP 的用户不产生 user_mfa 行。
func TestUserMFATOTPDataMigration(t *testing.T) {
	tdb := newTempCapabilityDB(t, "jimu_p16_data").TestDB
	caps := catalog.All()
	cfg := tdb.Config()

	require.NoError(t, db.Migrate(cfg, caps, "up"))
	require.NoError(t, db.Migrate(cfg, caps, "down")) // 回到 015：users 带 totp_*

	// 存量启用 TOTP 用户（totp_secret 直接写密文字符串，模拟既有 AES-GCM 值）；
	// PG 下 totp_enabled 是 SMALLINT，统一用 1/0 写入。
	require.NoError(t, tdb.DB.Exec(
		"INSERT INTO users (id, username, password, status, tenant_id, totp_secret, totp_enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,?,NOW(),NOW())",
		uint64(9001), "p16_mfa_user", "hash", 1, 1, "enc:v1:CIPHERTEXT", 1).Error)
	// 未启用用户：不应搬迁
	require.NoError(t, tdb.DB.Exec(
		"INSERT INTO users (id, username, password, status, tenant_id, totp_enabled, created_at, updated_at) VALUES (?,?,?,?,?,?,NOW(),NOW())",
		uint64(9002), "p16_mfa_user2", "hash", 1, 1, 0).Error)

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
}
