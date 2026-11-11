package db

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// versionedRow 测试用带乐观锁版本号的模型
type versionedRow struct {
	ID      uint64 `gorm:"primaryKey"`
	Name    string
	Version int64 `gorm:"not null;default:0"`
}

func (versionedRow) TableName() string { return "versioned_rows" }

func newConcurrencyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&versionedRow{}))
	return db
}

func TestSaveOptimisticUpdatesAndBumpsVersion(t *testing.T) {
	db := newConcurrencyTestDB(t)
	row := &versionedRow{Name: "a"}
	require.NoError(t, db.Create(row).Error)

	require.NoError(t, SaveOptimistic(db, &versionedRow{}, row.ID, row.Version, map[string]interface{}{"name": "b"}))

	var got versionedRow
	require.NoError(t, db.First(&got, row.ID).Error)
	assert.Equal(t, "b", got.Name)
	assert.Equal(t, int64(1), got.Version, "版本号应自增")
}

func TestSaveOptimisticDetectsConcurrentUpdate(t *testing.T) {
	db := newConcurrencyTestDB(t)
	row := &versionedRow{Name: "a"}
	require.NoError(t, db.Create(row).Error)

	// 两个请求读到同一版本
	first, second := row.Version, row.Version
	require.NoError(t, SaveOptimistic(db, &versionedRow{}, row.ID, first, map[string]interface{}{"name": "first"}))
	err := SaveOptimistic(db, &versionedRow{}, row.ID, second, map[string]interface{}{"name": "second"})
	assert.ErrorIs(t, err, ErrConcurrentUpdate, "版本不匹配应报并发冲突")

	var got versionedRow
	require.NoError(t, db.First(&got, row.ID).Error)
	assert.Equal(t, "first", got.Name, "后到的写入不应覆盖")
}

func TestSaveOptimisticRequiresFields(t *testing.T) {
	db := newConcurrencyTestDB(t)
	err := SaveOptimistic(db, &versionedRow{}, 1, 0, nil)
	assert.Error(t, err)
}

func TestLockRowUsesForUpdateOnMySQL(t *testing.T) {
	// DryRun 只构建 SQL，不真正连接数据库
	gdb, err := gorm.Open(
		mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true},
	)
	require.NoError(t, err)

	res := LockRow(gdb.Session(&gorm.Session{DryRun: true}), &versionedRow{}, 1)
	require.NoError(t, res.Error)
	assert.Contains(t, res.Statement.SQL.String(), "FOR UPDATE", "MySQL 应使用行锁")
}

func TestLockRowDegradesOnSQLite(t *testing.T) {
	db := newConcurrencyTestDB(t)
	row := &versionedRow{Name: "a"}
	require.NoError(t, db.Create(row).Error)

	// SQLite 不支持 FOR UPDATE，应降级为普通查询并正常返回行
	res := LockRow(db.Session(&gorm.Session{DryRun: true}), &versionedRow{}, row.ID)
	require.NoError(t, res.Error)
	assert.NotContains(t, res.Statement.SQL.String(), "FOR UPDATE")
}
