package infrastructure

import (
	"context"

	"jimu/internal/modules/audit/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *mysqlAuditRepository) FindByID(ctx context.Context, id uint64) (*domain.AuditLog, error) {
	var log domain.AuditLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *mysqlAuditRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
