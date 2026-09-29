package infrastructure

import (
	"context"

	"jimu/internal/capabilities/mfa/domain"

	"gorm.io/gorm"
)

type mysqlMFARepository struct {
	db *gorm.DB
}

// NewMysqlMFARepository 创建 TOTP 状态仓储
func NewMysqlMFARepository(db *gorm.DB) domain.MFARepository {
	return &mysqlMFARepository{db: db}
}

func (r *mysqlMFARepository) FindByUser(ctx context.Context, userID uint64) (*domain.UserMFA, error) {
	var record domain.UserMFA
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *mysqlMFARepository) UpsertSecret(ctx context.Context, tenantID, userID uint64, secret string, enabled bool) error {
	var existing domain.UserMFA
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(&domain.UserMFA{
			TenantID:    tenantID,
			UserID:      userID,
			TOTPSecret:  secret,
			TOTPEnabled: enabled,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&domain.UserMFA{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{"totp_secret": secret, "totp_enabled": enabled}).Error
}

func (r *mysqlMFARepository) Clear(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.UserMFA{}).Error
}
