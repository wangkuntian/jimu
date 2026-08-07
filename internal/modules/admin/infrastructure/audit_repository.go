package infrastructure

import (
	"context"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlAuditRepository struct {
	db *gorm.DB
}

// NewMysqlAuditRepository 创建审计日志 MySQL 仓储
func NewMysqlAuditRepository(db *gorm.DB) domain.AuditRepository {
	return &mysqlAuditRepository{db: db}
}

func (r *mysqlAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *mysqlAuditRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.AuditLog{})
	if adminID, ok := filters["admin_id"]; ok {
		query = query.Where("admin_id = ?", adminID)
	}
	if action, ok := filters["action"]; ok {
		query = query.Where("action = ?", action)
	}
	if resource, ok := filters["resource"]; ok {
		if resourceStr, ok := resource.(string); ok {
			query = query.Where("resource LIKE ?", "%"+resourceStr+"%")
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
