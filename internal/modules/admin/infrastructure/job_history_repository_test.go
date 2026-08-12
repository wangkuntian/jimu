package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/platform/queue/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&domain.JobHistory{}, &domain.DeadLetter{}, &domain.Job{}))
	return db
}

func TestMysqlJobHistoryRepositoryCreateAndList(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlJobHistoryRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.JobHistory{JobID: 1, Status: "success", Duration: 10})
	assert.NoError(t, err)

	history, err := repo.ListByJobID(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "success", history[0].Status)

	// 不存在的 job 返回空列表
	empty, err := repo.ListByJobID(ctx, 999)
	assert.NoError(t, err)
	assert.Len(t, empty, 0)
}
