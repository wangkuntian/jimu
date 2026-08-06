package application

import (
	"context"
	"strings"

	"jimu/internal/modules/admin/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
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

// UpdateUserRequest 管理员更新用户请求
type UpdateUserRequest struct {
	Status *int8    `json:"status" binding:"omitempty,oneof=0 1"`
	Roles  []string `json:"roles"`
}

// AdminUserService 用户管理服务
type AdminUserService struct {
	apiKeyRepo domain.APIKeyRepository
	auditRepo  domain.AuditRepository
}

// NewAdminUserService 创建用户管理服务
func NewAdminUserService() *AdminUserService {
	return &AdminUserService{}
}

// ValidateUserNotSelf 验证管理员不能操作自己
func (s *AdminUserService) ValidateUserNotSelf(adminID, targetID uint64) error {
	if adminID == targetID {
		return apperrors.New(apperrors.CodeForbidden, "cannot modify yourself")
	}
	return nil
}

// ValidateNotLastSuperAdmin 验证不会删除最后一个超级管理员
func (s *AdminUserService) ValidateNotLastSuperAdmin(ctx context.Context, targetID uint64) error {
	// TODO: implement count check when user repo is wired
	return nil
}

// SearchUsers 搜索用户（返回过滤后的用户列表）
func (s *AdminUserService) SearchUsers(ctx context.Context, filter ListUserFilter, p pagination.Pagination) ([]AdminUser, int64, error) {
	// TODO: implement with actual user repository
	return []AdminUser{}, 0, nil
}

// sanitizeSortField 防止 SQL 注入的排序字段白名单
func sanitizeSortField(sort string) string {
	allowed := map[string]bool{
		"id":         true,
		"username":   true,
		"status":     true,
		"created_at": true,
	}
	if allowed[sort] {
		return sort
	}
	return "id"
}

// containsRole 检查角色列表是否包含指定角色
func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if strings.EqualFold(r, target) {
			return true
		}
	}
	return false
}
