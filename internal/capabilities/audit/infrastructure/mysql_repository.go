package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"jimu/internal/capabilities/audit/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mysqlAuditRepository struct {
	db     *gorm.DB
	secret []byte // 审计链 HMAC 密钥（空则退化为 SHA-256）
}

// NewMysqlAuditRepository 创建审计仓储；hashSecret 用于链式哈希（建议非空）
func NewMysqlAuditRepository(db *gorm.DB, hashSecret string) domain.AuditRepository {
	return &mysqlAuditRepository{db: db, secret: []byte(hashSecret)}
}

// Create 写入单条审计并在事务内补齐链式哈希
func (r *mysqlAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	logs := []domain.AuditLog{*log}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := chainLogs(tx, logs, r.secret); err != nil {
			return err
		}
		return tx.Create(&logs).Error
	}); err != nil {
		return err
	}
	*log = logs[0]
	return nil
}

// CreateBatch 批量写入；先锁定链头再依次哈希，保证批次内顺序与全序一致
func (r *mysqlAuditRepository) CreateBatch(ctx context.Context, logs []domain.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := chainLogs(tx, logs, r.secret); err != nil {
			return err
		}
		return tx.Create(&logs).Error
	})
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
// CountRange 统计时间范围内的条目数（tenantID=0 表示平台级，不过滤租户）
func (r *mysqlAuditRepository) CountRange(ctx context.Context, tenantID uint64, start, end time.Time) (int64, error) {
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Where("created_at >= ? AND created_at < ?", start, end)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	err := db.Count(&total).Error
	return total, err
}

// ListRange 按时间范围分页返回条目（id 升序，保证分批导出稳定）
func (r *mysqlAuditRepository) ListRange(ctx context.Context, tenantID uint64, start, end time.Time, offset, limit int) ([]domain.AuditLog, error) {
	var logs []domain.AuditLog
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Where("created_at >= ? AND created_at < ?", start, end)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	err := db.Order("id ASC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, err
	}
	for i := range logs {
		deserializeChanges(&logs[i])
	}
	return logs, nil
}

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
