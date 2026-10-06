package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuditRetentionRulesOwnOnlyAuditLogs(t *testing.T) {
	rules := auditRetentionRules(RetentionConfig{AuditLogDays: 180})
	require.Len(t, rules, 1)
	assert.Equal(t, "audit_logs", rules[0].Table)
	assert.Equal(t, "created_at", rules[0].TimeColumn)
	assert.Equal(t, 180, rules[0].Days)
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
