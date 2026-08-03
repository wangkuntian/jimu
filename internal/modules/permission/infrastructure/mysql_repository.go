package infrastructure

import (
	"context"

	"jimu/internal/modules/role/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mysqlPermissionRepository struct {
	db *gorm.DB
}

func NewMysqlPermissionRepository(db *gorm.DB) domain.PermissionRepository {
	return &mysqlPermissionRepository{db: db}
}

func (r *mysqlPermissionRepository) FindByID(ctx context.Context, id uint64) (*domain.Permission, error) {
	var perm domain.Permission
	err := r.db.WithContext(ctx).First(&perm, id).Error
	return &perm, err
}

func (r *mysqlPermissionRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.Permission, int64, error) {
	var perms []domain.Permission
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Permission{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&perms).Error
	return perms, total, err
}

func (r *mysqlPermissionRepository) Create(ctx context.Context, perm *domain.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *mysqlPermissionRepository) Update(ctx context.Context, perm *domain.Permission) error {
	return r.db.WithContext(ctx).Save(perm).Error
}

func (r *mysqlPermissionRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Permission{}, id).Error
}
