package application

import (
	"context"
	"testing"

	importdomain "jimu/internal/capabilities/dataops/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// fakeImportJobRepo 可配置的导入任务仓储 mock
type fakeImportJobRepo struct {
	create   func(ctx context.Context, job *importdomain.ImportJob) error
	findByID func(ctx context.Context, id uint64) (*importdomain.ImportJob, error)
	update   func(ctx context.Context, job *importdomain.ImportJob) error
}

func (f *fakeImportJobRepo) Create(ctx context.Context, job *importdomain.ImportJob) error {
	if f.create != nil {
		return f.create(ctx, job)
	}
	job.ID = 42
	return nil
}

func (f *fakeImportJobRepo) FindByID(ctx context.Context, id uint64) (*importdomain.ImportJob, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &importdomain.ImportJob{ID: id, Type: "users", Status: importdomain.ImportJobCompleted}, nil
}

func (f *fakeImportJobRepo) Update(ctx context.Context, job *importdomain.ImportJob) error {
	if f.update != nil {
		return f.update(ctx, job)
	}
	return nil
}

// newSqliteDB 创建内存 sqlite 并迁移给定模型
func newSqliteDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	if len(models) > 0 {
		assert.NoError(t, db.AutoMigrate(models...))
	}
	return db
}
