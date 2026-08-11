package infrastructure

import (
	"context"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlImportJobRepository struct {
	db *gorm.DB
}

// NewMysqlImportJobRepository 创建导入任务 MySQL 仓储
func NewMysqlImportJobRepository(db *gorm.DB) domain.ImportJobRepository {
	return &mysqlImportJobRepository{db: db}
}

func (r *mysqlImportJobRepository) Create(ctx context.Context, job *domain.ImportJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *mysqlImportJobRepository) FindByID(ctx context.Context, id uint64) (*domain.ImportJob, error) {
	var job domain.ImportJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *mysqlImportJobRepository) Update(ctx context.Context, job *domain.ImportJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}
