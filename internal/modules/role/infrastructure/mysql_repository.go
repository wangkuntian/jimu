package infrastructure

import (
	"context"

	"jimu/internal/modules/role/domain"

	"gorm.io/gorm"
)

type mysqlRepository struct {
	db *gorm.DB
}

func NewMysqlRepository(db *gorm.DB) domain.RoleRepository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint64) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	return &role, err
}

func (r *mysqlRepository) FindAll(ctx context.Context) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).Find(&roles).Error
	return roles, err
}

func (r *mysqlRepository) Create(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *mysqlRepository) Update(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Role{}, id).Error
}

func (r *mysqlRepository) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&rolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			if err := tx.Create(&rolePermission{RoleID: roleID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *mysqlRepository) GetPermissions(ctx context.Context, roleID uint64) ([]domain.Permission, error) {
	var permissions []domain.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

type rolePermission struct {
	RoleID       uint64 `gorm:"primaryKey"`
	PermissionID uint64 `gorm:"primaryKey"`
}

func (rolePermission) TableName() string { return "role_permissions" }
