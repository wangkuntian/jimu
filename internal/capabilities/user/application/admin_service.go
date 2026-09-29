package application

import (
	"context"
	stderrors "errors"
	"strings"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminUser 管理端用户列表/详情响应（保持拆分前 admin 能力的响应形状）。
type AdminUser struct {
	ID        uint64   `json:"id"`
	Username  string   `json:"username"`
	Status    int8     `json:"status"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

// ListUserFilter 管理端用户列表过滤条件
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

// AdminUpdateUserRequest 管理员更新用户请求
type AdminUpdateUserRequest struct {
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	Email  *string `json:"email" binding:"omitempty,email"`
	Phone  *string `json:"phone" binding:"omitempty"`
}

// AdminUserService 用户管理面用例。与自助面（UserService）共享同一
// domain.UserRepository、租户可见性与配额校验，消除两处实现漂移。
type AdminUserService struct {
	userRepo domain.UserRepository
	users    *UserService     // 复用缓存失效与 outbox 事件（可 nil）
	roles    UserRoleAssigner // access 能力提供（可 nil）
	quota    TenantQuota      // tenant 能力提供（可 nil）
	db       *gorm.DB
}

// UserRoleAssigner 替换用户角色的端口（由 access 能力实现）。
// 只声明本能力用到的能力，避免跨能力依赖具体类型。
type UserRoleAssigner interface {
	AssignRoles(ctx context.Context, userID uint64, roleNames []string) error
}

// TenantQuota 租户配额校验（由 tenant 能力实现；nil = 未启用配额）。
type TenantQuota interface {
	CheckUserQuota(ctx context.Context, tenantID uint64) error
}

// NewAdminUserService 创建用户管理面服务。
func NewAdminUserService(userRepo domain.UserRepository, users *UserService) *AdminUserService {
	return &AdminUserService{userRepo: userRepo, users: users}
}

// WithRoles 注入用户角色分配端口（access 能力）。
func (s *AdminUserService) WithRoles(roles UserRoleAssigner) *AdminUserService {
	s.roles = roles
	return s
}

// WithQuota 注入租户配额校验（tenant 能力）。
func (s *AdminUserService) WithQuota(quota TenantQuota) *AdminUserService {
	s.quota = quota
	return s
}

// WithDB 注入事务用 db（角色分配失败回滚等场景可选）。
func (s *AdminUserService) WithDB(db *gorm.DB) *AdminUserService {
	s.db = db
	return s
}

// ListUsers 获取用户列表（搜索/过滤/分页；按上下文租户过滤，0=平台级视角不过滤）
func (s *AdminUserService) ListUsers(ctx context.Context, filter ListUserFilter, p pagination.Pagination) ([]AdminUser, int64, error) {
	users, total, err := s.userRepo.List(ctx, tenant.FromContext(ctx), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, "failed to list users", err)
	}
	result := make([]AdminUser, 0, len(users))
	for _, u := range users {
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
	if !tenantAllowed(user.TenantID, tenant.FromContext(ctx)) {
		return nil, apperrors.New(apperrors.CodeNotFound, "user not found")
	}
	return &AdminUser{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateUser 创建用户（配额校验在写入前完成；新用户归属创建者租户）
func (s *AdminUserService) CreateUser(ctx context.Context, req AdminCreateUserRequest) (*AdminUser, error) {
	tenantID := tenant.FromContext(ctx)
	if tenantID == 0 {
		tenantID = tenant.DefaultTenantID
	}
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
		TenantID: tenantID,
	}
	if req.Status == 0 {
		user.Status = 1
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeConflict, "failed to create user", err)
	}
	// 与自助面共享缓存失效与用户创建事件，避免两处漂移
	if s.users != nil {
		s.users.publishUserCreated(ctx, user)
	}
	return &AdminUser{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateUser 更新用户状态/联系方式（跨租户不可见）
func (s *AdminUserService) UpdateUser(ctx context.Context, id uint64, req AdminUpdateUserRequest) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	if !tenantAllowed(user.TenantID, tenant.FromContext(ctx)) {
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
	if err := s.userRepo.Update(ctx, user); err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to update user", err)
	}
	if s.users != nil {
		s.users.invalidateUserCache(ctx, id)
		s.users.publishUserUpdated(ctx, id)
	}
	return nil
}

// DisableUser 禁用用户
func (s *AdminUserService) DisableUser(ctx context.Context, id uint64) error {
	return s.UpdateUser(ctx, id, AdminUpdateUserRequest{Status: int8Ptr(0)})
}

// AssignRoles 为用户分配角色：委托 access 能力（user_roles 表所有者）。
func (s *AdminUserService) AssignRoles(ctx context.Context, userID uint64, roleNames []string) error {
	if s.roles == nil {
		return apperrors.New(apperrors.CodeInternalError, "role assignment is not configured")
	}
	return s.roles.AssignRoles(ctx, userID, roleNames)
}

// int8Ptr 返回 int8 指针
func int8Ptr(v int8) *int8 { return &v }
