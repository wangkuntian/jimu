package infrastructure

import (
	"context"
	"time"

	"jimu/internal/capabilities/auth/domain"

	"gorm.io/gorm"
)

type mysqlTrustedDeviceRepository struct {
	db *gorm.DB
}

// NewMysqlTrustedDeviceRepository 创建可信设备仓储
func NewMysqlTrustedDeviceRepository(db *gorm.DB) domain.TrustedDeviceRepository {
	return &mysqlTrustedDeviceRepository{db: db}
}

func (r *mysqlTrustedDeviceRepository) Create(ctx context.Context, device *domain.TrustedDevice) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *mysqlTrustedDeviceRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.TrustedDevice, error) {
	var device domain.TrustedDevice
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *mysqlTrustedDeviceRepository) Touch(ctx context.Context, id uint64, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.TrustedDevice{}).
		Where("id = ?", id).
		Update("last_used_at", usedAt).Error
}

func (r *mysqlTrustedDeviceRepository) ListByUser(ctx context.Context, tenantID, userID uint64) ([]domain.TrustedDevice, error) {
	var devices []domain.TrustedDevice
	db := r.db.WithContext(ctx).Model(&domain.TrustedDevice{}).Where("user_id = ?", userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	err := db.Order("id DESC").Find(&devices).Error
	return devices, err
}

func (r *mysqlTrustedDeviceRepository) Delete(ctx context.Context, tenantID, userID, id uint64) error {
	db := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return db.Delete(&domain.TrustedDevice{}).Error
}

func (r *mysqlTrustedDeviceRepository) DeleteAllByUser(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.TrustedDevice{}).Error
}

func (r *mysqlTrustedDeviceRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&domain.TrustedDevice{})
	return res.RowsAffected, res.Error
}
