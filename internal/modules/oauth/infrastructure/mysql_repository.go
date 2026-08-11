// internal/modules/oauth/infrastructure/mysql_repository.go
package infrastructure

import (
	"context"

	"jimu/internal/modules/oauth/domain"

	"gorm.io/gorm"
)

// MySQLBindingRepository 基于 MySQL 的绑定仓储
type MySQLBindingRepository struct {
	db *gorm.DB
}

// NewMySQLBindingRepository 创建绑定仓储
func NewMySQLBindingRepository(db *gorm.DB) *MySQLBindingRepository {
	return &MySQLBindingRepository{db: db}
}

// FindByProviderSubject 按 provider+subject 查绑定
func (r *MySQLBindingRepository) FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.OAuthBinding, error) {
	var binding domain.OAuthBinding
	err := r.db.WithContext(ctx).
		Where("provider = ? AND subject = ?", provider, subject).
		First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// Create 创建绑定
func (r *MySQLBindingRepository) Create(ctx context.Context, binding *domain.OAuthBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

var _ domain.BindingRepository = (*MySQLBindingRepository)(nil)
