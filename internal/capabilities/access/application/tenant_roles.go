package application

import (
	"context"

	"jimu/internal/capabilities/access/domain"
	kernelaccess "jimu/internal/kernel/access"
	dbutil "jimu/internal/kernel/db"
	apperrors "jimu/internal/shared/errors"

	"gorm.io/gorm"
)

// TenantRoleService 清理租户生命周期结束后的角色数据。
type TenantRoleService struct {
	db *gorm.DB
}

type rolePermissionLink struct {
	RoleID       uint64 `gorm:"primaryKey"`
	PermissionID uint64 `gorm:"primaryKey"`
}

func (rolePermissionLink) TableName() string { return "role_permissions" }

func NewTenantRoleService(db *gorm.DB) *TenantRoleService {
	return &TenantRoleService{db: db}
}

// DeleteTenantRoles 在同一事务内清理租户角色、权限关联和用户角色关联，再软删除角色。
func (s *TenantRoleService) DeleteTenantRoles(ctx context.Context, tenantID uint64) error {
	if s.db == nil {
		return apperrors.New(apperrors.CodeInternalError, "db not configured for tenant role cleanup")
	}
	_, externalTx := dbutil.TransactionFromContext(ctx)
	err := s.withTx(ctx, func(tx *gorm.DB) error {
		roleIDs := func() *gorm.DB {
			return tx.Table("roles").Select("id").Where("tenant_id = ?", tenantID)
		}
		userIDs := func() *gorm.DB {
			return tx.Table("users").Select("id").Where("tenant_id = ?", tenantID)
		}

		if err := tx.Where("role_id IN (?) OR user_id IN (?)", roleIDs(), userIDs()).
			Delete(&userRole{}).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to remove tenant user roles", err)
		}
		if err := tx.Where("role_id IN (?)", roleIDs()).
			Delete(&rolePermissionLink{}).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to remove tenant role permissions", err)
		}
		if err := tx.Where("tenant_id = ?", tenantID).Delete(&domain.Role{}).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to delete tenant roles", err)
		}
		return nil
	})
	if err == nil && !externalTx {
		kernelaccess.InvalidatePolicyCache()
	}
	return err
}

func (s *TenantRoleService) withTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if tx, ok := dbutil.TransactionFromContext(ctx); ok {
		return fn(tx.WithContext(ctx))
	}
	return s.db.WithContext(ctx).Transaction(fn)
}
