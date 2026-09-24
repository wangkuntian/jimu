package infrastructure

import (
	"context"

	"jimu/internal/capabilities/product/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mysqlProductRepository struct {
	db *gorm.DB
}

func NewMysqlProductRepository(db *gorm.DB) domain.ProductRepository {
	return &mysqlProductRepository{db: db}
}

func (r *mysqlProductRepository) FindByID(ctx context.Context, id uint64) (*domain.Product, error) {
	var entity domain.Product
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *mysqlProductRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.Product, int64, error) {
	var items []domain.Product
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Product{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *mysqlProductRepository) Create(ctx context.Context, entity *domain.Product) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *mysqlProductRepository) Update(ctx context.Context, entity *domain.Product) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *mysqlProductRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Product{}, id).Error
}
