package infrastructure

import (
	"context"
	"encoding/json"

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
	deserializeChanges(&log)
	return &log, nil
}

// List 分页查询审计日志。tenantID 非 0 时仅返回该租户的日志（0=平台级视角，不过滤）。
func (r *mysqlAuditRepository) List(ctx context.Context, tenantID uint64, offset, limit int, sort, order string) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{})
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range logs {
		deserializeChanges(&logs[i])
	}
	return logs, total, err
}

// deserializeChanges 从 ChangesRaw 反序列化字段变更列表
func deserializeChanges(log *domain.AuditLog) {
	if log.ChangesRaw == "" {
		return
	}
	_ = json.Unmarshal([]byte(log.ChangesRaw), &log.Changes)
}
