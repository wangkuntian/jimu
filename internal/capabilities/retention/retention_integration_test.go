package retention

import (
	"context"
	"testing"
	"time"

	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retentionProbeRow 集成测试专用表：避免耦合真实业务表结构
type retentionProbeRow struct {
	ID        uint64    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (retentionProbeRow) TableName() string { return "retention_probe" }

// TestRetentionServiceMySQLIntegration 针对真实 MySQL/MariaDB 验证分批删除 SQL。
// 重点：MySQL 不支持 IN (SELECT ... LIMIT n)，子查询必须包派生表；sqlite 测试无法覆盖该差异。
// CI 通过 services.mariadb 提供；本地无数据库时自动跳过。
func TestRetentionServiceMySQLIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	ctx := context.Background()
	require.NoError(t, tdb.DB.Exec("DROP TABLE IF EXISTS retention_probe").Error)
	require.NoError(t, tdb.DB.Exec(`CREATE TABLE retention_probe (
		id BIGINT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id)
	)`).Error)
	t.Cleanup(func() { _ = tdb.DB.Exec("DROP TABLE IF EXISTS retention_probe").Error })

	old := time.Now().AddDate(0, 0, -100).UTC().Truncate(time.Second)
	fresh := time.Now().UTC().Truncate(time.Second)
	for i := uint64(1); i <= 3; i++ {
		require.NoError(t, tdb.DB.Exec("INSERT INTO retention_probe (id, created_at) VALUES (?, ?)", i, old).Error)
	}
	require.NoError(t, tdb.DB.Exec("INSERT INTO retention_probe (id, created_at) VALUES (?, ?)", 99, fresh).Error)

	// BatchSize 2 触发多轮删除，覆盖派生表子查询语法
	svc := NewRetentionServiceWithRules(tdb.DB, []RetentionRule{
		{Table: "retention_probe", Model: &retentionProbeRow{}, TimeColumn: "created_at", Days: 30},
	}, 2)

	results, err := svc.Run(ctx)
	require.NoError(t, err, "MySQL 上分批删除 SQL 必须可执行")
	require.Len(t, results, 1)
	assert.Equal(t, int64(3), results[0].Deleted, "过期行应全部删除")

	var remaining []retentionProbeRow
	require.NoError(t, tdb.DB.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(99), remaining[0].ID, "未过期行应保留")
}
