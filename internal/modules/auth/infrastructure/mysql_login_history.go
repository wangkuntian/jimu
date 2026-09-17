package infrastructure

import (
	"context"

	"jimu/internal/modules/auth/domain"

	"gorm.io/gorm"
)

type mysqlLoginHistoryRepository struct {
	db *gorm.DB
}

// NewMysqlLoginHistoryRepository 创建登录历史仓储
func NewMysqlLoginHistoryRepository(db *gorm.DB) domain.LoginHistoryRepository {
	return &mysqlLoginHistoryRepository{db: db}
}

func (r *mysqlLoginHistoryRepository) Create(ctx context.Context, record *domain.LoginHistory) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *mysqlLoginHistoryRepository) ListByUser(ctx context.Context, tenantID, userID uint64, offset, limit int) ([]domain.LoginHistory, int64, error) {
	var records []domain.LoginHistory
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.LoginHistory{}).Where("user_id = ?", userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}
