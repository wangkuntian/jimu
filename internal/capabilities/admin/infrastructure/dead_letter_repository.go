package infrastructure

import (
	"context"
	"time"

	"jimu/internal/platform/queue/domain"

	"gorm.io/gorm"
)

type mysqlDeadLetterRepository struct {
	db *gorm.DB
}

// NewMysqlDeadLetterRepository 创建死信 MySQL 仓储
func NewMysqlDeadLetterRepository(db *gorm.DB) domain.DeadLetterRepository {
	return &mysqlDeadLetterRepository{db: db}
}

func (r *mysqlDeadLetterRepository) Create(ctx context.Context, d *domain.DeadLetter) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *mysqlDeadLetterRepository) List(ctx context.Context, tenantID uint64, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error) {
	var letters []domain.DeadLetter
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("resolved = ?", resolved)
	if tenantID != 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&letters).Error
	return letters, total, err
}

// MarkResolved 标记死信已处理；租户不匹配时不更新并返回 gorm.ErrRecordNotFound
func (r *mysqlDeadLetterRepository) MarkResolved(ctx context.Context, tenantID uint64, id uint64) error {
	query := r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("id = ?", id)
	if tenantID != 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	res := query.Updates(map[string]interface{}{"resolved": true, "resolved_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
