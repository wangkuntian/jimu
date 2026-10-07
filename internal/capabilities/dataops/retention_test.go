package dataops

import (
	"context"
	"testing"
	"time"

	"jimu/internal/capabilities/dataops/domain"
	"jimu/internal/shared/dbpurge"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDataopsRetentionDeletesOnlyCompletedImports(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.ImportJob{}))
	old := time.Now().AddDate(0, 0, -120)
	require.NoError(t, db.Create(&domain.ImportJob{ID: 1, Status: domain.ImportJobCompleted, CreatedAt: old}).Error)
	require.NoError(t, db.Create(&domain.ImportJob{ID: 2, Status: domain.ImportJobFailed, CreatedAt: old}).Error)
	require.NoError(t, db.Create(&domain.ImportJob{ID: 3, Status: domain.ImportJobProcessing, CreatedAt: old}).Error)

	results, err := dbpurge.New(db, 10).Run(context.Background(), dataopsRetentionRules(RetentionConfig{ImportJobDays: 90}))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(2), results[0].Deleted)
	var remaining []domain.ImportJob
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, uint64(3), remaining[0].ID)
}

func TestDataopsRetentionJobDefaultsAndGate(t *testing.T) {
	if _, ok := newDataopsRetentionJob(&gorm.DB{}, RetentionConfig{}, nil); ok {
		t.Fatal("disabled retention must not register a job")
	}
	if _, ok := newDataopsRetentionJob(nil, RetentionConfig{Enabled: true}, nil); ok {
		t.Fatal("retention without DB must not register a job")
	}
	job, ok := newDataopsRetentionJob(&gorm.DB{}, RetentionConfig{Enabled: true}, nil)
	require.True(t, ok)
	assert.Equal(t, "dataops_retention", job.ID)
	assert.Equal(t, "30 3 * * *", job.Spec)
}
