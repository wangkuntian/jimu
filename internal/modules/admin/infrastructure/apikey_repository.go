package infrastructure

import (
	"context"
	"time"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlAPIKeyRepository struct {
	db *gorm.DB
}

// NewMysqlAPIKeyRepository 创建 API Key MySQL 仓储
func NewMysqlAPIKeyRepository(db *gorm.DB) domain.APIKeyRepository {
	return &mysqlAPIKeyRepository{db: db}
}

func (r *mysqlAPIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *mysqlAPIKeyRepository) FindByID(ctx context.Context, id uint64) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.db.WithContext(ctx).First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *mysqlAPIKeyRepository) FindByKeyHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *mysqlAPIKeyRepository) List(ctx context.Context, offset, limit int) ([]domain.APIKey, int64, error) {
	var keys []domain.APIKey
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.APIKey{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(limit).Find(&keys).Error
	return keys, total, err
}

func (r *mysqlAPIKeyRepository) Update(ctx context.Context, key *domain.APIKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}

func (r *mysqlAPIKeyRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.APIKey{}, id).Error
}

func (r *mysqlAPIKeyRepository) IncrementUseCount(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&domain.APIKey{}).
		Where("id = ?", id).
		UpdateColumn("use_count", gorm.Expr("use_count + 1")).
		UpdateColumn("last_used", time.Now()).Error
}
