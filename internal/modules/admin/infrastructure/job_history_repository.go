package infrastructure

import (
	"context"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlJobHistoryRepository struct {
	db *gorm.DB
}

// NewMysqlJobHistoryRepository 创建任务历史 MySQL 仓储
func NewMysqlJobHistoryRepository(db *gorm.DB) domain.JobHistoryRepository {
	return &mysqlJobHistoryRepository{db: db}
}

func (r *mysqlJobHistoryRepository) Create(ctx context.Context, h *domain.JobHistory) error {
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *mysqlJobHistoryRepository) ListByJobID(ctx context.Context, jobID uint64) ([]domain.JobHistory, error) {
	var history []domain.JobHistory
	err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Order("id DESC").Find(&history).Error
	return history, err
}
