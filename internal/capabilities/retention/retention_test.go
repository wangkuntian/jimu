package retention

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newRetentionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&auditLogRow{}))
	return db
}

func TestRetentionServiceDeletesOnlyExpiredRows(t *testing.T) {
	db := newRetentionTestDB(t)
	ctx := context.Background()

	old := time.Now().AddDate(0, 0, -100)
	recent := time.Now().AddDate(0, 0, -1)
	require.NoError(t, db.Create(&auditLogRow{ID: 1, CreatedAt: old}).Error)
	require.NoError(t, db.Create(&auditLogRow{ID: 2, CreatedAt: recent}).Error)

	svc := NewRetentionService(db, Config{BatchSize: 100, AuditLogDays: 30})
	results, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "audit_logs", results[0].Table)
	assert.Equal(t, int64(1), results[0].Deleted)

	var remaining []auditLogRow
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(2), remaining[0].ID, "未过期行应保留")
}

func TestRetentionServiceDeletesInBatches(t *testing.T) {
	db := newRetentionTestDB(t)
	ctx := context.Background()

	old := time.Now().AddDate(0, 0, -100)
	for i := uint64(1); i <= 5; i++ {
		require.NoError(t, db.Create(&auditLogRow{ID: i, CreatedAt: old}).Error)
	}

	svc := NewRetentionService(db, Config{BatchSize: 2, AuditLogDays: 30})
	results, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(5), results[0].Deleted, "应分批删除全部过期行")

	var count int64
	require.NoError(t, db.Model(&auditLogRow{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestRetentionServiceSkipsDisabledTables(t *testing.T) {
	db := newRetentionTestDB(t)
	ctx := context.Background()

	require.NoError(t, db.Create(&auditLogRow{ID: 1, CreatedAt: time.Now().AddDate(0, 0, -100)}).Error)

	// 所有天数均为 0：不生成任何规则，不删除
	svc := NewRetentionService(db, Config{})
	results, err := svc.Run(ctx)
	require.NoError(t, err)
	assert.Empty(t, results)

	var count int64
	require.NoError(t, db.Model(&auditLogRow{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestRetentionServiceRespectsCondition(t *testing.T) {
	db := newRetentionTestDB(t)
	ctx := context.Background()

	old := time.Now().AddDate(0, 0, -100)
	require.NoError(t, db.Create(&auditLogRow{ID: 1, CreatedAt: old}).Error)
	require.NoError(t, db.Create(&auditLogRow{ID: 2, CreatedAt: old}).Error)

	svc := NewRetentionService(db, Config{})
	// 自定义规则：只清理 id=1
	svc.rules = []RetentionRule{{Table: "audit_logs", Model: &auditLogRow{}, TimeColumn: "created_at", Condition: "id = 1", Days: 30}}

	results, err := svc.Run(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)

	var remaining []auditLogRow
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(2), remaining[0].ID, "条件外的行应保留")
}

func TestDefaultRetentionRulesSkipZeroDays(t *testing.T) {
	rules := DefaultRetentionRules(Config{
		AuditLogDays:    180,
		OutboxEventDays: 7,
	})
	require.Len(t, rules, 2)
	assert.Equal(t, "audit_logs", rules[0].Table)
	assert.Equal(t, "outbox_events", rules[1].Table)
	// 死信用方言中立条件（MySQL/PG 通用）
	assert.Contains(t, DefaultRetentionRules(Config{DeadLetterDays: 30})[0].Condition, "TRUE")

	// 可信设备按过期时间清理（保留窗口内已失效的设备仍可用于排查）
	devices := DefaultRetentionRules(Config{TrustedDeviceDays: 7})
	require.Len(t, devices, 1)
	assert.Equal(t, "trusted_devices", devices[0].Table)
	assert.Equal(t, "expires_at", devices[0].TimeColumn)
}
