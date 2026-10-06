package mfa

import (
	"context"
	"testing"
	"time"

	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/shared/dbpurge"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMFARetentionKeepsRecentlyExpiredDevices(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&mfadomain.TrustedDevice{}))
	old := time.Now().AddDate(0, 0, -20)
	recent := time.Now().AddDate(0, 0, -3)
	require.NoError(t, db.Create(&mfadomain.TrustedDevice{ID: 1, TokenHash: "old", ExpiresAt: old}).Error)
	require.NoError(t, db.Create(&mfadomain.TrustedDevice{ID: 2, TokenHash: "recent", ExpiresAt: recent}).Error)

	rules := mfaRetentionRules(RetentionConfig{ExpiredDeviceDays: 7})
	require.Len(t, rules, 1)
	assert.Equal(t, "trusted_devices", rules[0].Table)
	assert.Equal(t, "expires_at", rules[0].TimeColumn)
	results, err := dbpurge.New(db, 10).Run(context.Background(), rules)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)
	var remaining []mfadomain.TrustedDevice
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(2), remaining[0].ID)
}

func TestMFARetentionJobDefaultsAndGate(t *testing.T) {
	if _, ok := newMFARetentionJob(&gorm.DB{}, RetentionConfig{}, nil); ok {
		t.Fatal("disabled retention must not register a job")
	}
	if _, ok := newMFARetentionJob(nil, RetentionConfig{Enabled: true}, nil); ok {
		t.Fatal("retention without DB must not register a job")
	}
	job, ok := newMFARetentionJob(&gorm.DB{}, RetentionConfig{Enabled: true}, nil)
	require.True(t, ok)
	assert.Equal(t, "mfa_retention", job.ID)
	assert.Equal(t, "30 3 * * *", job.Spec)
}
