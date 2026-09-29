package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/queue/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newQueueRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&domain.Job{}, &domain.DeadLetter{}))
	return db
}

func TestMysqlJobRepository(t *testing.T) {
	db := newQueueRepoTestDB(t)
	repo := NewMysqlJobRepository(db)
	ctx := context.Background()

	// Create + FindByID
	job := &domain.Job{ID: 1, Type: "email", Payload: "x", Status: domain.JobStatusPending, Priority: 5, MaxAttempts: 3}
	assert.NoError(t, repo.Create(ctx, job))
	got, err := repo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, domain.JobStatusPending, got.Status)

	// Update
	job.Status = domain.JobStatusFailed
	job.Attempts = 1
	job.Error = "boom"
	assert.NoError(t, repo.Update(ctx, job))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, domain.JobStatusFailed, got.Status)

	// List 带 status 过滤
	assert.NoError(t, repo.Create(ctx, &domain.Job{ID: 2, Type: "email", Status: domain.JobStatusPending}))
	jobs, total, err := repo.List(ctx, 0, 0, 10, map[string]interface{}{"status": "failed"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, uint64(1), jobs[0].ID)

	// List 带 type 过滤
	jobs, total, err = repo.List(ctx, 0, 0, 10, map[string]interface{}{"type": "sms"})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, jobs)

	// List 无过滤
	jobs, total, err = repo.List(ctx, 0, 0, 10, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)

	// List 按租户过滤
	assert.NoError(t, repo.Create(ctx, &domain.Job{ID: 3, TenantID: 5, Type: "email", Status: domain.JobStatusPending}))
	jobs, total, err = repo.List(ctx, 5, 0, 10, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, uint64(3), jobs[0].ID)
}
