package db

import (
	"context"
	"testing"

	"jimu/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newMockGormDB 用 sqlmock 构建 gorm.DB（跳过版本初始化，避免真实连接）
func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	dialector := mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return gormDB, mock
}

func TestDSN(t *testing.T) {
	cfg := config.DBConfig{Host: "db", Port: 3306, User: "u", Password: "p", Database: "app"}

	got := dsn(cfg, "", 0)
	want := "u:p@tcp(db:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, want, got)

	// 显式 host/port 优先
	got = dsn(cfg, "replica", 3307)
	want = "u:p@tcp(replica:3307)/app?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, want, got)

	// 只有 host 覆盖时 port 回落默认
	got = dsn(cfg, "replica", 0)
	want = "u:p@tcp(replica:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, want, got)
}

func TestPingDB(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectPing()
	require.NoError(t, pingDB(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigurePool(t *testing.T) {
	db, mock := newMockGormDB(t)
	cfg := config.DBConfig{
		MaxOpen:            10,
		MaxIdle:            5,
		ConnMaxLifetimeSec: 60,
		ConnMaxIdleTimeSec: 30,
	}
	configurePool(db, cfg)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, 10, sqlDB.Stats().MaxOpenConnections)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectWithRetry_ExhaustsRetries(t *testing.T) {
	// 指向 127.0.0.1:1，瞬时连接拒绝；MaxRetries=1 只重试一次
	cfg := config.DBConfig{
		Host: "127.0.0.1", Port: 1, User: "u", Password: "p", Database: "app",
		MaxRetries: 1, RetryIntervalSec: 1,
	}
	_, err := ConnectWithRetry(cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 1 attempts")
}

func TestConnectWithRetry_WithLogger(t *testing.T) {
	log, _ := newBufferLogger(t)
	cfg := config.DBConfig{
		Host: "127.0.0.1", Port: 1, User: "u", Password: "p", Database: "app",
		MaxRetries: 1, RetryIntervalSec: 1,
	}
	_, err := ConnectWithRetry(cfg, log)
	require.Error(t, err)
}

func TestNew_DelegatesToConnectWithRetry(t *testing.T) {
	cfg := config.DBConfig{
		Host: "127.0.0.1", Port: 1, User: "u", Password: "p", Database: "app",
		MaxRetries: 1, RetryIntervalSec: 1,
	}
	_, err := New(cfg, nil)
	require.Error(t, err)
}
