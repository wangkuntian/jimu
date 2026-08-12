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

func (r *mysqlDeadLetterRepository) List(ctx context.Context, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error) {
	var letters []domain.DeadLetter
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("resolved = ?", resolved)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&letters).Error
	return letters, total, err
}

func (r *mysqlDeadLetterRepository) MarkResolved(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("id = ?", id).
		Updates(map[string]interface{}{"resolved": true, "resolved_at": time.Now()}).Error
}
