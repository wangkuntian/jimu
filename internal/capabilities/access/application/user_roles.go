package application

import (
	"context"
	stderrors "errors"

	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"gorm.io/gorm"
)

// userRole 用户-角色关联（对应 user_roles 表，本能力为表所有者）。
type userRole struct {
	UserID uint64 `gorm:"primaryKey"`
	RoleID uint64 `gorm:"primaryKey"`
}

func (userRole) TableName() string { return "user_roles" }

// UserRoleService 用户角色分配用例（写 user_roles）。
type UserRoleService struct {
	db    *gorm.DB
	users userTenantReader // 只读用户归属，nil 时回退直查 users 表
}

// userTenantReader 读用户归属租户的最小端口（由 contract.UserinfoSource 适配）。
type userTenantReader interface {
	TenantOf(ctx context.Context, userID uint64) (uint64, error)
}

// NewUserRoleService 创建用户角色分配服务。
func NewUserRoleService(db *gorm.DB) *UserRoleService {
	return &UserRoleService{db: db}
}

// WithUsers 注入用户归属读取端口（可选）。
func (s *UserRoleService) WithUsers(users userTenantReader) *UserRoleService {
	s.users = users
	return s
}

// AssignRoles 用 roleNames 替换该用户的全部角色（事务内先锁用户行再替换）。
// 角色名在用户所属租户内解析；目标用户对上下文租户不可见时按不存在处理。
func (s *UserRoleService) AssignRoles(ctx context.Context, userID uint64, roleNames []string) error {
	if s.db == nil {
		return apperrors.New(apperrors.CodeInternalError, "db not configured for role assignment")
	}
	userTenant, err := s.userTenant(ctx, userID)
	if err != nil {
		return err
	}
	if !tenant.Visible(userTenant, tenant.FromContext(ctx)) {
		return apperrors.New(apperrors.CodeNotFound, "user not found")
	}

	// 解析角色名 -> 角色 ID（角色名租户内唯一，限定在用户所属租户内解析）
	var roleIDs []uint64
	if len(roleNames) > 0 {
		var roles []struct{ ID uint64 }
		query := s.db.WithContext(ctx).Table("roles").Where("name IN ?", roleNames)
		if userTenant != 0 {
			query = query.Where("tenant_id = ?", userTenant)
		}
		if err := query.Find(&roles).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to load roles", err)
		}
		for _, r := range roles {
			roleIDs = append(roleIDs, r.ID)
		}
		if len(roleIDs) != len(roleNames) {
			return apperrors.New(apperrors.CodeNotFound, "one or more roles not found")
		}
	}

	// 事务：先锁定用户行（串行化并发的角色替换），再清空旧角色、写入新角色
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dbutil.LockRow(tx, &userLock{}, userID).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.New(apperrors.CodeNotFound, "user not found")
			}
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to lock user", err)
		}
		if err := tx.Where("user_id = ?", userID).Delete(&userRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if err := tx.Create(&userRole{UserID: userID, RoleID: roleID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// userLock 仅用于事务内锁定 users 行（只带主键，不做用户领域逻辑）。
type userLock struct {
	ID uint64 `gorm:"primaryKey"`
}

func (userLock) TableName() string { return "users" }

// userTenant 读目标用户归属租户；用户不存在返回 404 语义错误。
func (s *UserRoleService) userTenant(ctx context.Context, userID uint64) (uint64, error) {
	if s.users != nil {
		tid, err := s.users.TenantOf(ctx, userID)
		if err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return 0, apperrors.New(apperrors.CodeNotFound, "user not found")
			}
			return 0, apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
		}
		return tid, nil
	}
	// 回退：直查 users 表（仅读 tenant_id，不做用户领域逻辑）
	var row struct {
		TenantID uint64 `gorm:"column:tenant_id"`
	}
	if err := s.db.WithContext(ctx).Table("users").Select("tenant_id").Where("id = ?", userID).Take(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperrors.New(apperrors.CodeNotFound, "user not found")
		}
		return 0, apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	return row.TenantID, nil
}
