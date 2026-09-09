package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/breaker"

	"github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBreakerConnPoolOpensOnConnectFailure(t *testing.T) {
	// 指向未监听端口：连接被拒绝，属于传输类失败
	sqlDB, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:1)/x?timeout=200ms")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	b := breaker.New("db-test", breaker.Config{MaxFailures: 2, ResetTimeout: time.Minute})
	pool := newBreakerConnPool(sqlDB, b)
	ctx := context.Background()

	query := func() error {
		_, err := pool.ExecContext(ctx, "SELECT 1")
		return err
	}

	_ = query()
	_ = query()
	assert.Equal(t, breaker.Open, b.State(), "连续连接失败应触发熔断")

	err = query()
	assert.ErrorIs(t, err, breaker.ErrOpen, "熔断开启后应快速失败")
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

func TestAttachBreakerSkipsWhenReadWriteSplit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	before := db.ConnPool
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: true}, true, nil))
	assert.Same(t, before, db.ConnPool, "读写分离时不应包装 ConnPool")
}

func TestAttachBreakerDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	before := db.ConnPool
	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: false}, false, nil))
	assert.Same(t, before, db.ConnPool)
}

func TestAttachBreakerWrapsSingleDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, attachBreaker(db, config.BreakerConfig{Enabled: true}, false, nil))
	_, ok := db.ConnPool.(*breakerConnPool)
	require.True(t, ok, "单库模式应包装 ConnPool")

	// 包装后 gorm.DB() 仍能取到底层连接池（健康检查/指标依赖）
	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.NotNil(t, sqlDB)

	// 正常查询与写入不受影响
	require.NoError(t, db.Exec("CREATE TABLE t (id INTEGER)").Error)
	require.NoError(t, db.Exec("INSERT INTO t (id) VALUES (1)").Error)
	var n int
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM t").Scan(&n).Error)
	assert.Equal(t, 1, n)
}
