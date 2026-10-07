package outbox

import (
	"context"
	"testing"
	"time"

	"jimu/internal/shared/dbpurge"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestOutboxRetentionPreservesUnpublishedEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Event{}))
	old := time.Now().AddDate(0, 0, -100)
	require.NoError(t, db.Create(&Event{ID: 1, CreatedAt: old, PublishedAt: &old}).Error)
	require.NoError(t, db.Create(&Event{ID: 2, CreatedAt: old, PublishedAt: nil}).Error)

	results, err := dbpurge.New(db, 10).Run(context.Background(), outboxRetentionRules(RetentionConfig{EventDays: 30}))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)
	var remaining []Event
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(2), remaining[0].ID)
}

func TestOutboxRetentionUsesPublishedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Event{}))
	old := time.Now().AddDate(0, 0, -100)
	recent := time.Now().AddDate(0, 0, -1)
	require.NoError(t, db.Create(&Event{ID: 1, CreatedAt: old, PublishedAt: &old}).Error)
	require.NoError(t, db.Create(&Event{ID: 2, CreatedAt: old, PublishedAt: &recent}).Error)

	results, err := dbpurge.New(db, 10).Run(context.Background(), outboxRetentionRules(RetentionConfig{EventDays: 30}))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0].Deleted)
	var remaining []Event
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(2), remaining[0].ID)
}

func TestOutboxRetentionJobDefaultsAndGate(t *testing.T) {
	if _, ok := newOutboxRetentionJob(&gorm.DB{}, RetentionConfig{}, nil); ok {
		t.Fatal("disabled retention must not register a job")
	}
	if _, ok := newOutboxRetentionJob(nil, RetentionConfig{Enabled: true}, nil); ok {
		t.Fatal("retention without DB must not register a job")
	}
	job, ok := newOutboxRetentionJob(&gorm.DB{}, RetentionConfig{Enabled: true}, nil)
	require.True(t, ok)
	assert.Equal(t, "outbox_retention", job.ID)
	assert.Equal(t, "30 3 * * *", job.Spec)
}
