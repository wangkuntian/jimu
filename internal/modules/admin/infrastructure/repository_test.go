package infrastructure

import (
	"context"
	"testing"

	admindomain "jimu/internal/modules/admin/domain"
	qdomain "jimu/internal/platform/queue/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(
		&admindomain.APIKey{},
		&admindomain.ImportJob{},
		&qdomain.Job{},
		&qdomain.DeadLetter{},
	))
	return db
}

func TestMysqlAPIKeyRepository(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewMysqlAPIKeyRepository(db)
	ctx := context.Background()

	// Create + FindByID
	key := &admindomain.APIKey{ID: 1, Name: "web", KeyPrefix: "jimu_ab", KeyHash: "h1", Scopes: "[\"read\"]", Enabled: true, CreatedBy: 3}
	assert.NoError(t, repo.Create(ctx, key))
	got, err := repo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, "web", got.Name)

	// FindByKeyHash
	byHash, err := repo.FindByKeyHash(ctx, "h1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), byHash.ID)

	// List
	keys, total, err := repo.List(ctx, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, int64(1), total)

	// Update
	key.Name = "web2"
	assert.NoError(t, repo.Update(ctx, key))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, "web2", got.Name)

	// IncrementUseCount
	assert.NoError(t, repo.IncrementUseCount(ctx, 1))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, int64(1), got.UseCount)

	// Delete
	assert.NoError(t, repo.Delete(ctx, 1))
	_, err = repo.FindByID(ctx, 1)
	assert.Error(t, err)
}

func TestMysqlImportJobRepository(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewMysqlImportJobRepository(db)
	ctx := context.Background()

	job := &admindomain.ImportJob{ID: 1, Type: "users", Filename: "a.csv", Status: admindomain.ImportJobProcessing, TotalRows: 5, CreatedBy: 1}
	assert.NoError(t, repo.Create(ctx, job))

	got, err := repo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, admindomain.ImportJobProcessing, got.Status)

	got.Status = admindomain.ImportJobCompleted
	got.SuccessRows = 5
	assert.NoError(t, repo.Update(ctx, got))
	got2, _ := repo.FindByID(ctx, 1)
	assert.Equal(t, admindomain.ImportJobCompleted, got2.Status)
	assert.Equal(t, 5, got2.SuccessRows)
}

func TestMysqlJobRepository(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewMysqlJobRepository(db)
	ctx := context.Background()

	// Create + FindByID
	job := &qdomain.Job{ID: 1, Type: "email", Payload: "x", Status: qdomain.JobStatusPending, Priority: 5, MaxAttempts: 3}
	assert.NoError(t, repo.Create(ctx, job))
	got, err := repo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, qdomain.JobStatusPending, got.Status)

	// Update
	job.Status = qdomain.JobStatusFailed
	job.Attempts = 1
	job.Error = "boom"
	assert.NoError(t, repo.Update(ctx, job))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, qdomain.JobStatusFailed, got.Status)

	// List 带 status 过滤
	assert.NoError(t, repo.Create(ctx, &qdomain.Job{ID: 2, Type: "email", Status: qdomain.JobStatusPending}))
	jobs, total, err := repo.List(ctx, 0, 10, map[string]interface{}{"status": "failed"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, uint64(1), jobs[0].ID)

	// List 带 type 过滤
	jobs, total, err = repo.List(ctx, 0, 10, map[string]interface{}{"type": "sms"})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, jobs)

	// List 无过滤
	jobs, total, err = repo.List(ctx, 0, 10, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)
}
