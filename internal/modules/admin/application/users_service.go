package application

import (
	"context"
	stderrors "errors"
	"strings"

	"jimu/internal/modules/user/domain"
	dbutil "jimu/internal/platform/db"
	"jimu/internal/platform/tenant"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// userRole 用户-角色关联（对应 user_roles 表）
type userRole struct {
	UserID uint64 `gorm:"primaryKey"`
	RoleID uint64 `gorm:"primaryKey"`
}

func (userRole) TableName() string { return "user_roles" }

// AdminUser DTO for admin user list responses
type AdminUser struct {
	ID        uint64   `json:"id"`
	Username  string   `json:"username"`
	Status    int8     `json:"status"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

// ListUserFilter 用户列表过滤条件
type ListUserFilter struct {
	Username      string
	Status        *int8
	RoleID        uint64
	CreatedAfter  string
	CreatedBefore string
}

// AdminCreateUserRequest 管理员创建用户请求
type AdminCreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=4,max=64"`
	Password string   `json:"password" binding:"required,min=8,max=32"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Phone    string   `json:"phone" binding:"omitempty"`
	Status   int8     `json:"status"`
	Roles    []string `json:"roles"`
}

// AdminUserService 用户管理服务
type AdminUserService struct {
	userRepo domain.UserRepository
	db       *gorm.DB
	quota    TenantQuota // nil = 未启用租户配额
}

// WithQuota 注入租户配额校验（未注入时不做配额检查）
func (s *AdminUserService) WithQuota(quota TenantQuota) *AdminUserService {
	s.quota = quota
	return s
}

// NewAdminUserService 创建用户管理服务
func NewAdminUserService(userRepo domain.UserRepository, db ...*gorm.DB) *AdminUserService {
	s := &AdminUserService{userRepo: userRepo}
	if len(db) > 0 {
		s.db = db[0]
	}
	return s
}

// AssignRoles 为用户分配角色（替换全部角色）
func (s *AdminUserService) AssignRoles(ctx context.Context, userID uint64, roleNames []string) error {
	if s.db == nil {
		return apperrors.New(apperrors.CodeInternalError, "db not configured for role assignment")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	if !tenantVisible(user.TenantID, tenant.FromContext(ctx)) {
		return apperrors.New(apperrors.CodeNotFound, "user not found")
	}

	// 解析角色名 -> 角色 ID（角色名租户内唯一，限定在用户所属租户内解析）
	var roleIDs []uint64
	if len(roleNames) > 0 {
		var roles []struct {
			ID uint64
		}
		query := s.db.WithContext(ctx).Table("roles").Where("name IN ?", roleNames)
		if user.TenantID != 0 {
			query = query.Where("tenant_id = ?", user.TenantID)
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
		var locked domain.User
		if err := dbutil.LockRow(tx, &locked, userID).Error; err != nil {
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

// ListUsers 获取用户列表（支持搜索/过滤/分页，按上下文租户过滤；0=平台级视角不过滤）
func (s *AdminUserService) ListUsers(ctx context.Context, filter ListUserFilter, p pagination.Pagination) ([]AdminUser, int64, error) {
	users, total, err := s.userRepo.List(ctx, tenant.FromContext(ctx), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, "failed to list users", err)
	}
	result := make([]AdminUser, 0, len(users))
	for _, u := range users {
		// 应用搜索过滤
		if filter.Username != "" && !strings.Contains(u.Username, filter.Username) {
			continue
		}
		if filter.Status != nil && u.Status != *filter.Status {
			continue
		}
		result = append(result, AdminUser{
			ID:        u.ID,
			Username:  u.Username,
			Status:    u.Status,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, total, nil
}

// GetUser 获取用户详情（跨租户不可见）
func (s *AdminUserService) GetUser(ctx context.Context, id uint64) (*AdminUser, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	if !tenantVisible(user.TenantID, tenant.FromContext(ctx)) {
		return nil, apperrors.New(apperrors.CodeNotFound, "user not found")
	}
	return &AdminUser{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateUser 创建用户
func (s *AdminUserService) CreateUser(ctx context.Context, req AdminCreateUserRequest) (*AdminUser, error) {
	// 新用户归属创建者所在租户；上下文无租户（平台级/旧 token）时归默认租户
	tenantID := tenant.FromContext(ctx)
	if tenantID == 0 {
		tenantID = tenant.DefaultTenantID
	}
	// 配额校验在写入前完成，超限直接拒绝（已存在的数据不受影响）
	if s.quota != nil {
		if err := s.quota.CheckUserQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to hash password", err)
	}
	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   req.Status,
	}
	user.TenantID = tenantID
	if req.Status == 0 {
		user.Status = 1
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeConflict, "failed to create user", err)
	}
	return &AdminUser{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// DisableUser 禁用用户
func (s *AdminUserService) DisableUser(ctx context.Context, id uint64) error {
	return s.UpdateUser(ctx, id, AdminUpdateUserRequest{Status: int8Ptr(0)})
}

// AdminUpdateUserRequest 管理员更新用户请求
type AdminUpdateUserRequest struct {
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	Email  *string `json:"email" binding:"omitempty,email"`
	Phone  *string `json:"phone" binding:"omitempty"`
}

// UpdateUser 更新用户状态/联系方式（Save 全字段保存，hook 重新加密并刷新盲索引）；跨租户不可见
func (s *AdminUserService) UpdateUser(ctx context.Context, id uint64, req AdminUpdateUserRequest) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	if !tenantVisible(user.TenantID, tenant.FromContext(ctx)) {
		return apperrors.New(apperrors.CodeNotFound, "user not found")
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	return s.userRepo.Update(ctx, user)
}

// int8Ptr 返回 int8 指针
func int8Ptr(v int8) *int8 {
	return &v
}
