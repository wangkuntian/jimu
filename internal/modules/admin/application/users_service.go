package application

import (
	"context"
	stderrors "errors"
	"strings"

	"jimu/internal/modules/user/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

// CreateUserRequest 管理员创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=4,max=64"`
	Password string   `json:"password" binding:"required,min=8,max=32"`
	Status   int8     `json:"status"`
	Roles    []string `json:"roles"`
}

// AdminUserService 用户管理服务
type AdminUserService struct {
	userRepo domain.UserRepository
}

// NewAdminUserService 创建用户管理服务
func NewAdminUserService(userRepo domain.UserRepository) *AdminUserService {
	return &AdminUserService{userRepo: userRepo}
}

// ValidateUserNotSelf 验证管理员不能操作自己
func (s *AdminUserService) ValidateUserNotSelf(adminID, targetID uint64) error {
	if adminID == targetID {
		return apperrors.New(apperrors.CodeForbidden, "cannot modify yourself")
	}
	return nil
}

// ListUsers 获取用户列表（支持搜索/过滤/分页）
func (s *AdminUserService) ListUsers(ctx context.Context, filter ListUserFilter, p pagination.Pagination) ([]AdminUser, int64, error) {
	users, total, err := s.userRepo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
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

// GetUser 获取用户详情
func (s *AdminUserService) GetUser(ctx context.Context, id uint64) (*AdminUser, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	return &AdminUser{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateUser 创建用户
func (s *AdminUserService) CreateUser(ctx context.Context, req CreateUserRequest) (*AdminUser, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to hash password", err)
	}
	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Status:   req.Status,
	}
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
	return s.UpdateUser(ctx, id, UpdateUserRequest{Status: int8Ptr(0)})
}

// UpdateUserRequest 管理员更新用户请求
type UpdateUserRequest struct {
	Status *int8 `json:"status" binding:"omitempty,oneof=0 1"`
}

// UpdateUser 更新用户状态
func (s *AdminUserService) UpdateUser(ctx context.Context, id uint64, req UpdateUserRequest) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.Wrap(apperrors.CodeNotFound, "user not found", err)
		}
		return apperrors.Wrap(apperrors.CodeInternalError, "failed to get user", err)
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	return s.userRepo.Update(ctx, user)
}

// int8Ptr 返回 int8 指针
func int8Ptr(v int8) *int8 {
	return &v
}
