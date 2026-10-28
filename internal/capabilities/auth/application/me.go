package application

import (
	"context"
	stderrors "errors"
	"time"

	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/shared/errors"
)

// MeUser 当前会话用户的只读视图。
type MeUser struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Status    int8      `json:"status"`
	TenantID  uint64    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MeView 会话引导所需的全部信息：身份 + 角色（仅供展示）+ 有效权限点（用于判定）。
type MeView struct {
	User        MeUser            `json:"user"`
	Roles       []string          `json:"roles"`
	Permissions []PermissionPoint `json:"permissions"`
}

// Me 返回当前用户身份与有效权限点。
//
// tenantID 来自已验签 claim（gin context，由 auth.AuthMiddleware 写入），
// 不取账号行里的 tenant_id —— 后者可能滞后于签发时的归属。
// 权限点经 kernel/access 的既有查询（角色 + 策略）由 EffectivePermissions 聚合。
func (s *AuthService) Me(ctx context.Context, userID, tenantID uint64) (*MeView, error) {
	account, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		// 只有「确实不存在」才映射 404；其余错误（如 DB 故障）必须保留为内部错误，
		// 否则一次瞬时故障会被会话引导误报成「账号已删除」。
		if stderrors.Is(err, contract.ErrNotFound) {
			return nil, errors.Wrap(errors.CodeUserNotFound, "user not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to load user", err)
	}

	// 两个切片都初始化为空切片：响应的 roles/permissions 必须是 [] 而不是 null
	roles := []string{}
	roleSubjects := []string{}
	policies := []access.Policy{}
	if s.authz != nil {
		identities, err := s.authz.RolesForUser(ctx, userID)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to load roles", err)
		}
		for _, identity := range identities {
			roles = append(roles, identity.Name)
			roleSubjects = append(roleSubjects, access.RoleSubject(identity.ID))
		}
		if policies, err = s.authz.Policies(ctx); err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to load policies", err)
		}
	}

	return &MeView{
		User: MeUser{
			ID:        account.ID,
			Username:  account.Username,
			Status:    account.Status,
			TenantID:  tenantID,
			CreatedAt: account.CreatedAt,
		},
		Roles:       roles,
		Permissions: EffectivePermissions(roleSubjects, policies),
	}, nil
}
