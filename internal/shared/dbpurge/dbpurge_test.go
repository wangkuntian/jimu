package dbpurge

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type purgeTestRow struct {
	ID        uint64    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
	Status    string
}

func (purgeTestRow) TableName() string { return "purge_test_rows" }

func newPurgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&purgeTestRow{}))
	return db
}

func TestServiceDeletesExpiredRowsInBatches(t *testing.T) {
	db := newPurgeTestDB(t)
	old := time.Now().AddDate(0, 0, -100)
	recent := time.Now().AddDate(0, 0, -1)
	for i := uint64(1); i <= 5; i++ {
		require.NoError(t, db.Create(&purgeTestRow{ID: i, CreatedAt: old, Status: "done"}).Error)
	}
	require.NoError(t, db.Create(&purgeTestRow{ID: 99, CreatedAt: recent, Status: "done"}).Error)

	results, err := New(db, 2).Run(context.Background(), []Rule{{
		Table: "purge_test_rows", Model: &purgeTestRow{}, TimeColumn: "created_at", Days: 30,
	}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(5), results[0].Deleted)

	var rows []purgeTestRow
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, uint64(99), rows[0].ID)
}

func TestServiceHonorsRuleConditionAndZeroDays(t *testing.T) {
	db := newPurgeTestDB(t)
	old := time.Now().AddDate(0, 0, -100)
	require.NoError(t, db.Create(&purgeTestRow{ID: 1, CreatedAt: old, Status: "done"}).Error)
	require.NoError(t, db.Create(&purgeTestRow{ID: 2, CreatedAt: old, Status: "active"}).Error)

	results, err := New(db, 10).Run(context.Background(), []Rule{
		{Table: "purge_test_rows", Model: &purgeTestRow{}, TimeColumn: "created_at", Condition: "status = ?", Args: []any{"done"}, Days: 30},
		{Table: "purge_test_rows", Model: &purgeTestRow{}, TimeColumn: "created_at", Days: 0},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)

	var rows []purgeTestRow
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, "active", rows[0].Status)
}

func TestServiceRequiresDatabase(t *testing.T) {
	_, err := New(nil, 10).Run(context.Background(), []Rule{{
		Table: "purge_test_rows", Model: &purgeTestRow{}, TimeColumn: "created_at", Days: 1,
	}})
	require.ErrorContains(t, err, "database")
}
