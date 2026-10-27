package application

import (
	"context"
	stderrors "errors"
	"strconv"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	kernelaccess "jimu/internal/kernel/access"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type RoleService struct {
	repo  domain.RoleRepository
	quota TenantQuota // nil = 未启用租户配额
}

func NewRoleService(repo domain.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

// WithQuota 注入租户配额校验（未注入时不做配额检查）
func (s *RoleService) WithQuota(quota TenantQuota) *RoleService {
	s.quota = quota
	return s
}

func (s *RoleService) Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error) {
	tenantID, err := s.createTenantID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	role := &domain.Role{Name: req.Name, Description: req.Description, Status: 1}
	role.TenantID = tenantID
	if s.quota != nil {
		if err := s.quota.CheckRoleQuota(ctx, role.TenantID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Create(ctx, role); err != nil {
		if isDuplicateKey(err) {
			return nil, errors.Wrap(errors.CodeConflict, "role already exists", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create role", err)
	}
	resp := ToRoleResponse(*role)
	return &resp, nil
}

func (s *RoleService) createTenantID(ctx context.Context, rawTenantID string) (uint64, error) {
	if tenant.IsPlatformScope(ctx) {
		if rawTenantID == "" {
			return 0, errors.New(errors.CodeInvalidParam, "tenant_id is required for platform roles")
		}
		id, err := strconv.ParseUint(rawTenantID, 10, 64)
		if err != nil || id == 0 {
			return 0, errors.New(errors.CodeInvalidParam, "invalid tenant_id")
		}
		if s.quota == nil {
			return 0, errors.New(errors.CodeInternalError, "tenant lookup is not configured")
		}
		exists, err := s.quota.TenantExists(ctx, id)
		if err != nil {
			return 0, errors.Wrap(errors.CodeInternalError, "failed to load tenant", err)
		}
		if !exists {
			return 0, errors.New(errors.CodeNotFound, "tenant not found")
		}
		return id, nil
	}

	ctxTenantID := tenant.FromContext(ctx)
	if rawTenantID != "" {
		id, err := strconv.ParseUint(rawTenantID, 10, 64)
		if err != nil || id == 0 {
			return 0, errors.New(errors.CodeInvalidParam, "invalid tenant_id")
		}
		if ctxTenantID != 0 && id != ctxTenantID {
			return 0, errors.New(errors.CodeForbidden, "tenant scope mismatch")
		}
		return id, nil
	}
	if ctxTenantID != 0 {
		return ctxTenantID, nil
	}
	return tenant.DefaultTenantID, nil
}

func (s *RoleService) Get(ctx context.Context, id uint64) (*RoleResponse, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !roleTenantAllowed(ctx, role.TenantID) {
		return nil, errors.New(errors.CodeNotFound, "role not found")
	}
	resp := ToRoleResponse(*role)
	return &resp, nil
}

// List 列出角色；filter 的零值表示平台级不过滤。
func (s *RoleService) List(ctx context.Context, p pagination.Pagination, filter domain.RoleListFilter) ([]RoleResponse, int64, error) {
	roles, total, err := s.repo.List(ctx, filter, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list roles", err)
	}
	return ToRoleResponses(roles), total, nil
}

// Options 返回目标租户内可用于用户角色分配的启用角色摘要。
// 平台管理员查看默认租户时同时返回平台角色。
func (s *RoleService) Options(ctx context.Context, requestedTenantID uint64) ([]RoleOptionResponse, error) {
	tenantID := tenant.FromContext(ctx)
	if tenant.IsPlatformScope(ctx) {
		if requestedTenantID != 0 {
			if s.quota != nil {
				exists, err := s.quota.TenantExists(ctx, requestedTenantID)
				if err != nil {
					return nil, errors.Wrap(errors.CodeInternalError, "failed to load tenant", err)
				}
				if !exists {
					return nil, errors.New(errors.CodeNotFound, "tenant not found")
				}
			}
			tenantID = requestedTenantID
		}
	} else if requestedTenantID != 0 && requestedTenantID != tenantID {
		return nil, errors.New(errors.CodeForbidden, "tenant scope mismatch")
	}
	if tenantID == 0 {
		return []RoleOptionResponse{}, nil
	}
	includePlatform := tenant.IsPlatformScope(ctx) && tenantID == tenant.DefaultTenantID
	roles, err := s.repo.ListOptions(ctx, tenantID, includePlatform)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to list role options", err)
	}
	out := make([]RoleOptionResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, ToRoleOptionResponse(role))
	}
	return out, nil
}

func (s *RoleService) Update(ctx context.Context, id uint64, req UpdateRoleRequest) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !roleTenantAllowed(ctx, role.TenantID) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	if role.IsSystem {
		return errors.New(errors.CodeForbidden, "system role is immutable")
	}
	role.Name = req.Name
	role.Description = req.Description
	if req.Status != nil {
		role.Status = *req.Status
	}
	if err := s.repo.Update(ctx, role); err != nil {
		if isDuplicateKey(err) {
			return errors.Wrap(errors.CodeConflict, "role already exists", err)
		}
		if stderrors.Is(err, dbutil.ErrConcurrentUpdate) {
			return errors.Wrap(errors.CodeConflict, "role was modified concurrently, please retry", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update role", err)
	}
	kernelaccess.InvalidatePolicyCache()
	return nil
}

func (s *RoleService) Delete(ctx context.Context, id uint64) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !roleTenantAllowed(ctx, role.TenantID) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	if role.IsSystem {
		return errors.New(errors.CodeForbidden, "system role is immutable")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		if stderrors.Is(err, domain.ErrRoleInUse) {
			return errors.New(errors.CodeConflict, "role is assigned to users")
		}
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to delete role", err)
	}
	kernelaccess.InvalidatePolicyCache()
	return nil
}

func (s *RoleService) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	role, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !roleTenantAllowed(ctx, role.TenantID) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	if role.IsSystem {
		return errors.New(errors.CodeForbidden, "system role is immutable")
	}
	permissionIDs = uniqueIDs(permissionIDs)
	if role.Scope != contract.RoleScopePlatform {
		permissions, err := s.repo.FindPermissionsByIDs(ctx, permissionIDs)
		if err != nil {
			return errors.Wrap(errors.CodeInternalError, "failed to load permissions", err)
		}
		if len(permissions) != len(permissionIDs) {
			return errors.New(errors.CodeInvalidParam, "one or more permissions not found")
		}
		for _, permission := range permissions {
			if contract.IsPlatformOnlyPermission(permission.Resource, permission.Action) {
				return errors.New(errors.CodeForbidden, "tenant role cannot include platform-only permissions")
			}
		}
	}
	if err := s.repo.AssignPermissions(ctx, roleID, permissionIDs); err != nil {
		if stderrors.Is(err, domain.ErrPermissionNotFound) {
			return errors.New(errors.CodeInvalidParam, "one or more permissions not found")
		}
		return errors.Wrap(errors.CodeInternalError, "failed to assign permissions", err)
	}
	kernelaccess.InvalidatePolicyCache()
	return nil
}

func (s *RoleService) GetPermissions(ctx context.Context, roleID uint64) ([]PermissionResponse, error) {
	role, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !roleTenantAllowed(ctx, role.TenantID) {
		return nil, errors.New(errors.CodeNotFound, "role not found")
	}
	permissions, err := s.repo.GetPermissions(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get permissions", err)
	}
	return ToPermissionResponses(permissions), nil
}

// roleTenantAllowed 判断目标角色是否允许当前上下文访问。
// 平台范围放行所有租户；其他上下文仅允许当前租户或未归属资源。
func roleTenantAllowed(ctx context.Context, resourceTenant uint64) bool {
	if tenant.IsPlatformScope(ctx) {
		return true
	}
	ctxTenant := tenant.FromContext(ctx)
	return ctxTenant == 0 || resourceTenant == 0 || resourceTenant == ctxTenant
}
