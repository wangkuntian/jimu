package infrastructure

import (
	"context"

	"jimu/internal/capabilities/queue/domain"
	"jimu/internal/kernel/tenant"

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

// ListByJobID 按任务 ID 查询执行历史；有租户上下文时按租户过滤（tid=0 为平台级视角，不过滤）。
func (r *mysqlJobHistoryRepository) ListByJobID(ctx context.Context, jobID uint64) ([]domain.JobHistory, error) {
	var history []domain.JobHistory
	db := r.db.WithContext(ctx).Where("job_id = ?", jobID)
	if tid := tenant.FromContext(ctx); tid != 0 {
		db = db.Where("tenant_id = ?", tid)
	}
	err := db.Order("id DESC").Find(&history).Error
	return history, err
}
