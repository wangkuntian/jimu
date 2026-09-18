package infrastructure

import (
	"context"

	"jimu/internal/capabilities/role/domain"
	dbutil "jimu/internal/platform/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// List 分页查询角色。tenantID 非 0 时仅返回该租户下的角色（0=平台级视角，不过滤）。
func (r *mysqlRepository) List(ctx context.Context, tenantID uint64, offset, limit int, sort, order string) ([]domain.Role, int64, error) {
	var roles []domain.Role
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Role{})
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&roles).Error
	return roles, total, err
}

func (r *mysqlRepository) Create(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// Update 乐观锁更新：version 不匹配时返回 db.ErrConcurrentUpdate（调用方映射 409）
func (r *mysqlRepository) Update(ctx context.Context, role *domain.Role) error {
	if err := dbutil.SaveOptimistic(r.db.WithContext(ctx), &domain.Role{}, role.ID, role.Version, map[string]interface{}{
		"tenant_id":   role.TenantID,
		"name":        role.Name,
		"description": role.Description,
		"status":      role.Status,
	}); err != nil {
		return err
	}
	role.Version++
	return nil
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Role{}, id).Error
}

// AssignPermissions 整体替换角色权限。事务内先锁定角色行，
// 避免并发调用交错导致「后写的删掉先写的授权」。
func (r *mysqlRepository) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role domain.Role
		if err := dbutil.LockRow(tx, &role, roleID).Error; err != nil {
			return err
		}
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
