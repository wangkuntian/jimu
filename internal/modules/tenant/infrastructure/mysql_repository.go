package infrastructure

import (
	"context"

	"jimu/internal/modules/tenant/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mysqlRepository struct {
	db *gorm.DB
}

func NewMysqlRepository(db *gorm.DB) domain.TenantRepository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint64) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *mysqlRepository) FindByCode(ctx context.Context, code string) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *mysqlRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.Tenant, int64, error) {
	var tenants []domain.Tenant
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Tenant{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&tenants).Error
	return tenants, total, err
}

func (r *mysqlRepository) Create(ctx context.Context, t *domain.Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *mysqlRepository) Update(ctx context.Context, t *domain.Tenant) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Tenant{}, id).Error
}
