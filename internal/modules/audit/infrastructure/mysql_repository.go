package infrastructure

import (
	"context"

	"jimu/internal/modules/audit/domain"

	"gorm.io/gorm"
)

type mysqlAuditRepository struct {
	db *gorm.DB
}

func NewMysqlAuditRepository(db *gorm.DB) domain.AuditRepository {
	return &mysqlAuditRepository{db: db}
}

func (r *mysqlAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *mysqlAuditRepository) CreateBatch(ctx context.Context, logs []domain.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&logs).Error
}

func (r *mysqlAuditRepository) List(ctx context.Context, offset, limit int) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{})
	db.Count(&total)
	err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
