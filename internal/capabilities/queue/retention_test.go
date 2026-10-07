package queue

import (
	"context"
	"testing"
	"time"

	"jimu/internal/capabilities/queue/domain"
	"jimu/internal/shared/dbpurge"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestQueueRetentionDeletesOnlyOwnedEligibleRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Job{}, &domain.JobHistory{}, &domain.DeadLetter{}))
	old := time.Now().AddDate(0, 0, -100)
	require.NoError(t, db.Create(&domain.Job{ID: 1, Status: domain.JobStatusSuccess, UpdatedAt: old}).Error)
	require.NoError(t, db.Create(&domain.Job{ID: 2, Status: domain.JobStatusPending, UpdatedAt: old}).Error)
	require.NoError(t, db.Create(&domain.JobHistory{ID: 1, EndedAt: old}).Error)
	require.NoError(t, db.Create(&domain.DeadLetter{ID: 1, Resolved: true, ResolvedAt: old}).Error)
	require.NoError(t, db.Create(&domain.DeadLetter{ID: 2, Resolved: false, ResolvedAt: old}).Error)

	results, err := dbpurge.New(db, 2).Run(context.Background(), queueRetentionRules(RetentionConfig{
		JobDays: 30, JobHistoryDays: 30, DeadLetterDays: 30,
	}))
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, int64(1), results[0].Deleted)
	assert.Equal(t, int64(1), results[1].Deleted)
	assert.Equal(t, int64(1), results[2].Deleted)

	var jobs []domain.Job
	require.NoError(t, db.Find(&jobs).Error)
	require.Len(t, jobs, 1)
	assert.Equal(t, uint64(2), jobs[0].ID)
	var deadLetters []domain.DeadLetter
	require.NoError(t, db.Find(&deadLetters).Error)
	require.Len(t, deadLetters, 1)
	assert.Equal(t, uint64(2), deadLetters[0].ID)
}

func TestQueueRetentionJobDefaultsAndGate(t *testing.T) {
	if _, ok := newQueueRetentionJob(&gorm.DB{}, RetentionConfig{}, nil); ok {
		t.Fatal("disabled retention must not register a job")
	}
	if _, ok := newQueueRetentionJob(nil, RetentionConfig{Enabled: true}, nil); ok {
		t.Fatal("retention without DB must not register a job")
	}
	job, ok := newQueueRetentionJob(&gorm.DB{}, RetentionConfig{Enabled: true}, nil)
	require.True(t, ok)
	assert.Equal(t, "queue_retention", job.ID)
	assert.Equal(t, "30 3 * * *", job.Spec)
}
