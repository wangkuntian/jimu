package infrastructure

import (
	"context"

	"jimu/internal/modules/role/domain"

	"gorm.io/gorm"
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

func (r *mysqlPermissionRepository) FindAll(ctx context.Context) ([]domain.Permission, error) {
	var perms []domain.Permission
	err := r.db.WithContext(ctx).Find(&perms).Error
	return perms, err
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
