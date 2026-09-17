package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/breaker"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func TestAttachBreakerOpensOnConnectFailure(t *testing.T) {
	// 指向未监听端口：连接被拒绝，属于传输类失败
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "root:root@tcp(127.0.0.1:1)/x?timeout=200ms",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true}) // 不预连：让失败发生在语句执行时
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: true, MaxFailures: 2, ResetTimeoutSec: 60}))

	query := func() error { return db.Exec("SELECT 1").Error }
	_ = query()
	_ = query()
	// 直接验证熔断状态：实际状态由插件内部的 breaker 持有，这里通过第三次调用的错误判断
	err = query()
	assert.ErrorIs(t, err, breaker.ErrOpen, "连续连接失败后熔断开启，应快速失败而不是继续等超时")
}

func TestIsDBTransportError(t *testing.T) {
	assert.False(t, isDBTransportError(nil))
	assert.False(t, isDBTransportError(sql.ErrNoRows))
	// 慢查询超时不熔断，避免误判数据库不可用
	assert.False(t, isDBTransportError(context.DeadlineExceeded))
	assert.True(t, isDBTransportError(driver.ErrBadConn))
	assert.True(t, isDBTransportError(sql.ErrConnDone))
	assert.True(t, isDBTransportError(errors.New("dial tcp 127.0.0.1:3306: connect: connection refused")))
	assert.True(t, isDBTransportError(errors.New("Error 1040: Too many connections")))
}

func TestAttachBreakerKeepsConnPoolIntact(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	before := db.ConnPool
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: true, MaxFailures: 5, ResetTimeoutSec: 10}))

	// 回调式挂载不接管 ConnPool：连接池上限、DB() 等对 *sql.DB 的操作不受影响
	assert.Same(t, before, db.ConnPool)

	// 正常查询与写入不受影响
	require.NoError(t, db.Exec("CREATE TABLE t (id INTEGER)").Error)
	require.NoError(t, db.Exec("INSERT INTO t (id) VALUES (1)").Error)
	var n int
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM t").Scan(&n).Error)
	assert.Equal(t, 1, n)

	// 业务错误（无匹配行）不应触发失败计数：连续多次查询仍可正常执行
	for i := 0; i < 10; i++ {
		var row struct{ ID int }
		err := db.Raw("SELECT id FROM t WHERE id = 999").Scan(&row).Error
		assert.NoError(t, err)
	}
	require.NoError(t, db.Exec("INSERT INTO t (id) VALUES (2)").Error)
}

func TestAttachBreakerDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	before := db.ConnPool
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: false}))
	assert.Same(t, before, db.ConnPool)
	assert.Nil(t, db.Callback().Query().Get("jimu:breaker:allow:query"), "未启用时不应注册熔断回调")
}

// TestAttachBreakerCoversReadReplicas 验证回调式熔断在读写分离下同样生效：
// dbresolver 为每个副本建独立连接池，语句级回调不受影响。
func TestAttachBreakerCoversReadReplicas(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.db")
	replica := mysql.New(mysql.Config{
		DSN:                       "root:root@tcp(127.0.0.1:1)/x?timeout=100ms",
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(sqlite.Open(source), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  []gorm.Dialector{sqlite.Open(source)},
		Replicas: []gorm.Dialector{replica},
		Policy:   dbresolver.RandomPolicy{},
	})))
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: true, MaxFailures: 2, ResetTimeoutSec: 60}))

	// 建表走主库（sqlite）
	require.NoError(t, db.Exec("CREATE TABLE breaker_rows (id INTEGER)").Error)

	// 读走副本（不可达）：连续失败触发熔断
	for i := 0; i < 2; i++ {
		var rows []breakerRow
		assert.Error(t, db.Find(&rows).Error)
	}

	// 熔断开启后读请求被语句级回调拦截（错误为 ErrOpen，而不是连接失败）
	var rows []breakerRow
	assert.ErrorIs(t, db.Find(&rows).Error, breaker.ErrOpen,
		"副本路径的失败也应驱动熔断")
}

// breakerRow 供读写分离测试使用的简单模型
type breakerRow struct {
	ID int
}
