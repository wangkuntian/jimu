package dbpurge

import (
	"context"
	"testing"
	"time"

	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mysqlPurgeProbeRow struct {
	ID        uint64    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (mysqlPurgeProbeRow) TableName() string { return "dbpurge_probe" }

func TestServiceMySQLIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.DB.Exec("DROP TABLE IF EXISTS dbpurge_probe").Error)
	require.NoError(t, tdb.DB.Exec(`CREATE TABLE dbpurge_probe (
		id BIGINT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id)
	)`).Error)
	t.Cleanup(func() { _ = tdb.DB.Exec("DROP TABLE IF EXISTS dbpurge_probe").Error })

	old := time.Now().AddDate(0, 0, -100).UTC().Truncate(time.Second)
	fresh := time.Now().UTC().Truncate(time.Second)
	for i := uint64(1); i <= 3; i++ {
		require.NoError(t, tdb.DB.Exec("INSERT INTO dbpurge_probe (id, created_at) VALUES (?, ?)", i, old).Error)
	}
	require.NoError(t, tdb.DB.Exec("INSERT INTO dbpurge_probe (id, created_at) VALUES (?, ?)", 99, fresh).Error)

	results, err := New(tdb.DB, 2).Run(context.Background(), []Rule{{
		Table: "dbpurge_probe", Model: &mysqlPurgeProbeRow{}, TimeColumn: "created_at", Days: 30,
	}})
	require.NoError(t, err, "MySQL must support the derived-table batch delete")
	require.Len(t, results, 1)
	assert.Equal(t, int64(3), results[0].Deleted)

	var remaining []mysqlPurgeProbeRow
	require.NoError(t, tdb.DB.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(99), remaining[0].ID)
}
