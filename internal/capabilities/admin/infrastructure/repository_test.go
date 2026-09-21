package infrastructure

import (
	"context"
	"testing"

	admindomain "jimu/internal/capabilities/admin/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(
		&admindomain.ImportJob{},
	))
	return db
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
