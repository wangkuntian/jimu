package scheduler

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// testDB 提供内存 SQLite 测试数据库
func testDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&jobModel{}); err != nil {
		return nil, err
	}
	return db, nil
}

func TestMySQLStore(t *testing.T) {
	db, err := testDB()
	assert.NoError(t, err)
	store := NewMySQLStore(db)
	ctx := context.Background()

	job := JobDef{ID: "job1", Name: "Test", Cron: "@every 1m", Enabled: true}
	assert.NoError(t, store.Save(ctx, job))

	list, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}
