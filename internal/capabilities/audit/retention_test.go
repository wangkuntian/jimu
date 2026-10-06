package audit

import (
	"context"
	"testing"
	"time"

	"jimu/internal/capabilities/audit/domain"
	"jimu/internal/shared/dbpurge"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuditRetentionRulesOwnOnlyAuditLogs(t *testing.T) {
	rules := auditRetentionRules(RetentionConfig{AuditLogDays: 180})
	require.Len(t, rules, 1)
	assert.Equal(t, "audit_logs", rules[0].Table)
	assert.Equal(t, "created_at", rules[0].TimeColumn)
	assert.Equal(t, "entry_hash = ''", rules[0].Condition)
	assert.Equal(t, 180, rules[0].Days)
}

func TestAuditRetentionPreservesHashedEntries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.AuditLog{}))
	old := time.Now().AddDate(0, 0, -200)
	require.NoError(t, db.Create(&domain.AuditLog{ID: 1, CreatedAt: old}).Error)
	require.NoError(t, db.Create(&domain.AuditLog{ID: 2, CreatedAt: old, EntryHash: "hashed"}).Error)

	results, err := dbpurge.New(db, 10).Run(context.Background(), auditRetentionRules(RetentionConfig{AuditLogDays: 180}))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)
	var remaining []domain.AuditLog
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, "hashed", remaining[0].EntryHash)
}

func TestNewAuditRetentionJobHonorsEnabledAndDatabase(t *testing.T) {
	db := &gorm.DB{}
	if _, ok := newAuditRetentionJob(db, RetentionConfig{Cron: "30 3 * * *"}, nil); ok {
		t.Fatal("disabled retention must not register a job")
	}
	if _, ok := newAuditRetentionJob(nil, RetentionConfig{Enabled: true, Cron: "30 3 * * *"}, nil); ok {
		t.Fatal("retention without a database must not register a job")
	}
	job, ok := newAuditRetentionJob(db, RetentionConfig{Enabled: true, Cron: "30 3 * * *"}, nil)
	require.True(t, ok)
	assert.Equal(t, "audit_retention", job.ID)
	assert.Equal(t, "30 3 * * *", job.Spec)
}
