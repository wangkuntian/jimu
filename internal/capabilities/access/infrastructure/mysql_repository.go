package infrastructure

import (
	"context"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/shared/sqlpattern"

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

// List 分页查询角色。filter 的零值表示平台级不过滤。
func (r *mysqlRepository) List(ctx context.Context, filter domain.RoleListFilter, offset, limit int, sort, order string) ([]domain.Role, int64, error) {
	var roles []domain.Role
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Role{})
	if filter.TenantID != 0 {
		db = db.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Keyword != "" {
		db = db.Where("name LIKE ? ESCAPE '"+sqlpattern.EscapeChar+"'", sqlpattern.LikeContains(filter.Keyword))
	}
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
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

func (r *mysqlRepository) ListOptions(ctx context.Context, tenantID uint64) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = 1 AND scope = ?", tenantID, contract.RoleScopeTenant).
		Order("is_system desc, id asc").
		Find(&roles).Error
	return roles, err
}

func (r *mysqlRepository) FindPermissionsByIDs(ctx context.Context, ids []uint64) ([]domain.Permission, error) {
	if len(ids) == 0 {
		return []domain.Permission{}, nil
	}
	var permissions []domain.Permission
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&permissions).Error
	return permissions, err
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

// AssignPermissions 整体替换角色权限。事务内先锁定角色行，
// 避免并发调用交错导致「后写的删掉先写的授权」。
func (r *mysqlRepository) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role domain.Role
		if err := dbutil.LockRow(tx, &role, roleID).Error; err != nil {
			return err
		}
		if len(permissionIDs) > 0 {
			var count int64
			if err := tx.Model(&domain.Permission{}).
				Where("id IN ? AND deleted_at IS NULL", permissionIDs).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(permissionIDs)) {
				return domain.ErrPermissionNotFound
			}
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
