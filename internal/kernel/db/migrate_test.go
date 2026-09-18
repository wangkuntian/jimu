package db

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationDir(t *testing.T) {
	dir := MigrationDir()
	assert.True(t, filepath.IsAbs(dir))
	assert.Equal(t, "mysql", filepath.Base(dir))
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFindUp(t *testing.T) {
	root := t.TempDir()
	migDir := filepath.Join(root, "migrations")
	require.NoError(t, os.MkdirAll(migDir, 0o755))
	deep := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0o755))

	assert.Equal(t, migDir, findUp(deep, "migrations"))
	assert.Equal(t, migDir, findUp(root, "migrations"))
	// 从 target 目录自身开始查找
	assert.Equal(t, migDir, findUp(migDir, "migrations"))
	// 不存在时返回空串
	assert.Equal(t, "", findUp(deep, "nope"))
}

func TestIsDir(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "f.txt")
	require.NoError(t, os.WriteFile(file, nil, 0o644))

	assert.True(t, isDir(root))
	assert.False(t, isDir(file))
	assert.False(t, isDir(filepath.Join(root, "missing")))
}

func TestRunMigration_DirectionsFailWithoutDB(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	cfg := config.DBConfig{}
	// 各方向分支都会触达 goose，但对 sqlmock 无预期查询 → 返回错误。
	// 断言有错误即覆盖 switch 各分支与 goose 调用路径。
	for _, dir := range []string{"up", "up-by-one", "down", "redo", "status", "reset"} {
		t.Run(dir, func(t *testing.T) {
			err := runMigration(sqlDB, cfg, dir)
			require.Error(t, err)
		})
	}
}

func TestRunMigration_UnknownDirection(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	err = runMigration(sqlDB, config.DBConfig{}, "bogus")
	require.ErrorContains(t, err, "unknown direction: bogus")
}

func TestMigrate_UnknownDirectionNoDBContact(t *testing.T) {
	cfg := config.DBConfig{Host: "127.0.0.1", Port: 3306, User: "u", Password: "p", Database: "app"}
	err := Migrate(cfg, "bogus")
	require.ErrorContains(t, err, "unknown direction: bogus")
}

func TestMigrateWithRetry_ExhaustsRetries(t *testing.T) {
	// "bogus" 方向在 runMigration 立即失败，不触达真实 DB
	cfg := config.DBConfig{Host: "127.0.0.1", Port: 3306, User: "u", Password: "p", Database: "app",
		MaxRetries: 1, RetryIntervalSec: 1}
	err := MigrateWithRetry(cfg, nil, "bogus")
	require.ErrorContains(t, err, "failed after 1 attempts")
}

func TestMigrateWithRetry_WithLogger(t *testing.T) {
	log, _ := newBufferLogger(t)
	cfg := config.DBConfig{Host: "127.0.0.1", Port: 3306, User: "u", Password: "p", Database: "app",
		MaxRetries: 1, RetryIntervalSec: 1}
	err := MigrateWithRetry(cfg, log, "bogus")
	require.Error(t, err)
}

func TestAutoMigrate_FailsOnMockDB(t *testing.T) {
	db, _ := newMockGormDB(t)
	err := AutoMigrate(db, &cleanupModel{})
	require.Error(t, err)
}
