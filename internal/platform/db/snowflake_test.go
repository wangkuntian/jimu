// internal/platform/db/snowflake_test.go
package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type snowflakeModel struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

// openTestDB 建 sqlite 内存库并注册雪花 hook
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	RegisterSnowflakeHook(db)
	require.NoError(t, db.AutoMigrate(&snowflakeModel{}))
	return db
}

func TestSnowflakeHook_AssignsID(t *testing.T) {
	require.NoError(t, InitSnowflake(1))
	db := openTestDB(t)
	t.Cleanup(func() { snowflakeGen = nil })

	m := snowflakeModel{Name: "a"}
	require.NoError(t, db.Create(&m).Error)
	// 雪花 ID 在 ms 时间戳 << 22 量级，远大于自增小整数
	assert.Greater(t, m.ID, uint64(1)<<40)

	// 从库读取，ID 持久化
	var got snowflakeModel
	require.NoError(t, db.First(&got, "name = ?", "a").Error)
	assert.Equal(t, m.ID, got.ID)

	// 再次创建，ID 单调递增且不重复
	var m2 snowflakeModel
	require.NoError(t, db.Create(&m2).Error)
	assert.NotEqual(t, m.ID, m2.ID)
	assert.Greater(t, m2.ID, m.ID)
}

func TestSnowflakeHook_NoOverwriteExistingID(t *testing.T) {
	require.NoError(t, InitSnowflake(2))
	db := openTestDB(t)
	t.Cleanup(func() { snowflakeGen = nil })

	m := snowflakeModel{ID: 42, Name: "explicit"}
	require.NoError(t, db.Create(&m).Error)
	assert.Equal(t, uint64(42), m.ID)
}

func TestSnowflakeHook_BatchCreate(t *testing.T) {
	require.NoError(t, InitSnowflake(3))
	db := openTestDB(t)
	t.Cleanup(func() { snowflakeGen = nil })

	ms := []snowflakeModel{{Name: "x"}, {Name: "y"}, {Name: "z"}}
	require.NoError(t, db.Create(&ms).Error)
	for _, m := range ms {
		assert.Greater(t, m.ID, uint64(1)<<40, "batch 元素应获得雪花 ID")
	}
	// 两两不同
	assert.NotEqual(t, ms[0].ID, ms[1].ID)
	assert.NotEqual(t, ms[1].ID, ms[2].ID)
}

func TestSnowflakeHook_NoopWithoutInit(t *testing.T) {
	snowflakeGen = nil
	db := openTestDB(t)

	// 未 InitSnowflake 时回退自增（sqlite 自增从 1 开始）
	m := snowflakeModel{Name: "auto"}
	require.NoError(t, db.Create(&m).Error)
	assert.Equal(t, uint64(1), m.ID)
}
