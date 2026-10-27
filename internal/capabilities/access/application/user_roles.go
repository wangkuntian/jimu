package application

import (
	"context"
	stderrors "errors"
	"sort"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// AssignRoleIDs 用 roleIDs 替换该用户的全部角色（事务内先锁用户行再替换）。
// 角色必须在用户所属租户内；目标用户对上下文租户不可见时按不存在处理。
func (s *UserRoleService) AssignRoleIDs(ctx context.Context, userID uint64, roleIDs []uint64) error {
	if s.db == nil {
		return apperrors.New(apperrors.CodeInternalError, "db not configured for role assignment")
	}
	roleIDs = uniqueIDs(roleIDs)
	db := s.dbFor(ctx)
	userTenant, err := s.userTenant(ctx, userID)
	if err != nil {
		return err
	}
	if !tenant.IsPlatformScope(ctx) && userTenant == tenant.DefaultTenantID {
		return apperrors.New(apperrors.CodeNotFound, "user not found")
	}
	if !tenant.IsPlatformScope(ctx) && !tenant.Visible(userTenant, tenant.FromContext(ctx)) {
		return apperrors.New(apperrors.CodeNotFound, "user not found")
	}

	// 校验角色 ID 存在且属于用户所属租户。
	if len(roleIDs) > 0 {
		var roles []struct{ ID uint64 }
		query := db.Table("roles").Where("id IN ? AND deleted_at IS NULL AND status = 1", roleIDs)
		if userTenant != 0 {
			query = query.Where("tenant_id = ?", userTenant)
		}
		if err := query.Find(&roles).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to load roles", err)
		}
		if len(roles) != len(roleIDs) {
			return apperrors.New(apperrors.CodeNotFound, "one or more roles not found")
		}
	}

	// 事务：先锁定用户行（串行化并发的角色替换），再清空旧角色、写入新角色
	return s.withTx(ctx, func(tx *gorm.DB) error {
		if err := s.ensureNotRemovingLastSuperAdmin(tx, userID, roleIDs); err != nil {
			return err
		}
		if err := dbutil.LockRow(tx, &userLock{}, userID).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.New(apperrors.CodeNotFound, "user not found")
			}
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to lock user", err)
		}
		if err := s.lockAssignableRoles(tx, userTenant, roleIDs); err != nil {
			return err
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

func (s *UserRoleService) lockAssignableRoles(tx *gorm.DB, userTenant uint64, roleIDs []uint64) error {
	ordered := append([]uint64(nil), roleIDs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for _, roleID := range ordered {
		var role domain.Role
		if err := dbutil.LockRow(tx, &role, roleID).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.New(apperrors.CodeNotFound, "one or more roles not found")
			}
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to lock role", err)
		}
		if role.Status != 1 || (userTenant != 0 && role.TenantID != userTenant) {
			return apperrors.New(apperrors.CodeNotFound, "one or more roles not found")
		}
	}
	return nil
}

func (s *UserRoleService) EnsureCanDeactivateUser(ctx context.Context, userID uint64) error {
	if s.db == nil {
		return apperrors.New(apperrors.CodeInternalError, "db not configured for role assignment")
	}
	return s.withTx(ctx, func(tx *gorm.DB) error {
		return s.ensureNotRemovingLastSuperAdmin(tx, userID, nil)
	})
}

func (s *UserRoleService) ensureNotRemovingLastSuperAdmin(tx *gorm.DB, userID uint64, nextRoleIDs []uint64) error {
	var role domain.Role
	query := tx.Where(
		"tenant_id = ? AND scope = ? AND code = ? AND status = 1 AND deleted_at IS NULL",
		tenant.DefaultTenantID, contract.RoleScopePlatform, contract.RoleCodePlatformSuperAdmin,
	)
	if tx.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to lock platform superadmin role", err)
	}

	active, err := activePlatformSuperAdmin(tx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to inspect platform superadmin", err)
	}
	if !active {
		return nil
	}
	if len(nextRoleIDs) > 0 {
		var nextCount int64
		if err := tx.Table("roles").
			Where("id IN ? AND code = ? AND status = 1 AND deleted_at IS NULL", nextRoleIDs, contract.RoleCodePlatformSuperAdmin).
			Count(&nextCount).Error; err != nil {
			return apperrors.Wrap(apperrors.CodeInternalError, "failed to inspect next roles", err)
		}
		if nextCount > 0 {
			return nil
		}
	}

	var otherCount int64
	if err := tx.Table("users").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.id <> ? AND users.status = 1 AND users.deleted_at IS NULL", userID).
		Where("roles.tenant_id = ? AND roles.scope = ? AND roles.status = 1 AND roles.deleted_at IS NULL AND roles.code = ?",
			tenant.DefaultTenantID, contract.RoleScopePlatform, contract.RoleCodePlatformSuperAdmin).
		Distinct("users.id").
		Count(&otherCount).Error; err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to count platform superadmins", err)
	}
	if otherCount == 0 {
		return apperrors.New(apperrors.CodeConflict, "cannot remove the last active platform superadmin")
	}
	return nil
}

func activePlatformSuperAdmin(tx *gorm.DB, userID uint64) (bool, error) {
	var count int64
	err := tx.Table("users").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.id = ? AND users.status = 1 AND users.deleted_at IS NULL", userID).
		Where("roles.tenant_id = ? AND roles.scope = ? AND roles.status = 1 AND roles.deleted_at IS NULL AND roles.code = ?",
			tenant.DefaultTenantID, contract.RoleScopePlatform, contract.RoleCodePlatformSuperAdmin).
		Count(&count).Error
	return count > 0, err
}

// RolesForUsers 批量读取用户当前角色。
func (s *UserRoleService) RolesForUsers(ctx context.Context, userIDs []uint64) (map[uint64][]contract.UserRole, error) {
	out := make(map[uint64][]contract.UserRole, len(userIDs))
	if s.db == nil || len(userIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID   uint64
		RoleID   uint64
		RoleName string
	}
	err := s.dbFor(ctx).
		Table("user_roles ur").
		Select("ur.user_id, roles.id AS role_id, roles.name AS role_name").
		Joins("JOIN roles ON roles.id = ur.role_id AND roles.deleted_at IS NULL AND roles.status = 1").
		Where("ur.user_id IN ?", userIDs).
		Order("ur.user_id, roles.id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserID] = append(out[row.UserID], contract.UserRole{ID: row.RoleID, Name: row.RoleName})
	}
	return out, nil
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
	if err := s.dbFor(ctx).Table("users").Select("tenant_id").Where("id = ?", userID).Take(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperrors.New(apperrors.CodeNotFound, "user not found")
		}
		return 0, apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	return row.TenantID, nil
}

func (s *UserRoleService) dbFor(ctx context.Context) *gorm.DB {
	if tx, ok := dbutil.TransactionFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

func (s *UserRoleService) withTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if tx, ok := dbutil.TransactionFromContext(ctx); ok {
		return fn(tx.WithContext(ctx))
	}
	return s.db.WithContext(ctx).Transaction(fn)
}

func uniqueIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
