package infrastructure

import (
	"context"
	"time"

	apikeydomain "jimu/internal/capabilities/apikey/domain"

	"gorm.io/gorm"
)

// mysqlAPIKeyRepository 读写 api_keys 表；表实体归 apikey 能力（设计 §5.2），
// 仓储接口随模型留在 apikey/domain（P1.7 拆分 admin 时再收口）。
type mysqlAPIKeyRepository struct {
	db *gorm.DB
}

// NewMysqlAPIKeyRepository 创建 API Key MySQL 仓储
func NewMysqlAPIKeyRepository(db *gorm.DB) apikeydomain.APIKeyRepository {
	return &mysqlAPIKeyRepository{db: db}
}

func (r *mysqlAPIKeyRepository) Create(ctx context.Context, key *apikeydomain.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *mysqlAPIKeyRepository) FindByID(ctx context.Context, id uint64) (*apikeydomain.APIKey, error) {
	var key apikeydomain.APIKey
	err := r.db.WithContext(ctx).First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *mysqlAPIKeyRepository) FindByKeyHash(ctx context.Context, hash string) (*apikeydomain.APIKey, error) {
	var key apikeydomain.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *mysqlAPIKeyRepository) List(ctx context.Context, tenantID uint64, offset, limit int) ([]apikeydomain.APIKey, int64, error) {
	var keys []apikeydomain.APIKey
	var total int64
	db := r.db.WithContext(ctx).Model(&apikeydomain.APIKey{})
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&keys).Error
	return keys, total, err
}

func (r *mysqlAPIKeyRepository) Update(ctx context.Context, key *apikeydomain.APIKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}

func (r *mysqlAPIKeyRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&apikeydomain.APIKey{}, id).Error
}

func (r *mysqlAPIKeyRepository) IncrementUseCount(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&apikeydomain.APIKey{}).
		Where("id = ?", id).
		UpdateColumn("use_count", gorm.Expr("use_count + 1")).
		UpdateColumn("last_used", time.Now()).Error
}
