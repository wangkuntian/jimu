package infrastructure

import (
	"context"

	"jimu/internal/platform/queue/domain"

	"gorm.io/gorm"
)

type mysqlJobRepository struct {
	db *gorm.DB
}

// NewMysqlJobRepository 创建任务 MySQL 仓储
func NewMysqlJobRepository(db *gorm.DB) domain.JobRepository {
	return &mysqlJobRepository{db: db}
}

func (r *mysqlJobRepository) Create(ctx context.Context, job *domain.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *mysqlJobRepository) FindByID(ctx context.Context, id uint64) (*domain.Job, error) {
	var job domain.Job
	err := r.db.WithContext(ctx).First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *mysqlJobRepository) Update(ctx context.Context, job *domain.Job) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *mysqlJobRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]domain.Job, int64, error) {
	var jobs []domain.Job
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Job{})
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if jobType, ok := filters["type"]; ok {
		query = query.Where("type = ?", jobType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&jobs).Error
	return jobs, total, err
}
