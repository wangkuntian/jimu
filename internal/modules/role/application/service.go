package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/role/domain"
	"jimu/internal/platform/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type RoleService struct {
	repo domain.RoleRepository
}

func NewRoleService(repo domain.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error) {
	role := &domain.Role{Name: req.Name, Description: req.Description, Status: 1}
	// 角色归属创建者所在租户；上下文无租户（平台级/旧 token）时归默认租户
	role.TenantID = tenant.FromContext(ctx)
	if role.TenantID == 0 {
		role.TenantID = tenant.DefaultTenantID
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

func (s *RoleService) Get(ctx context.Context, id uint64) (*RoleResponse, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !tenantAllowed(role.TenantID, tenant.FromContext(ctx)) {
		return nil, errors.New(errors.CodeNotFound, "role not found")
	}
	resp := ToRoleResponse(*role)
	return &resp, nil
}

func (s *RoleService) List(ctx context.Context, p pagination.Pagination) ([]RoleResponse, int64, error) {
	roles, total, err := s.repo.List(ctx, tenant.FromContext(ctx), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list roles", err)
	}
	return ToRoleResponses(roles), total, nil
}

func (s *RoleService) Update(ctx context.Context, id uint64, req UpdateRoleRequest) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !tenantAllowed(role.TenantID, tenant.FromContext(ctx)) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	role.Name = req.Name
	role.Description = req.Description
	if err := s.repo.Update(ctx, role); err != nil {
		if isDuplicateKey(err) {
			return errors.Wrap(errors.CodeConflict, "role already exists", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update role", err)
	}
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
	if !tenantAllowed(role.TenantID, tenant.FromContext(ctx)) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete role", err)
	}
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
	if !tenantAllowed(role.TenantID, tenant.FromContext(ctx)) {
		return errors.New(errors.CodeNotFound, "role not found")
	}
	return s.repo.AssignPermissions(ctx, roleID, permissionIDs)
}

func (s *RoleService) GetPermissions(ctx context.Context, roleID uint64) ([]PermissionResponse, error) {
	role, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "role not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get role", err)
	}
	if !tenantAllowed(role.TenantID, tenant.FromContext(ctx)) {
		return nil, errors.New(errors.CodeNotFound, "role not found")
	}
	permissions, err := s.repo.GetPermissions(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get permissions", err)
	}
	return ToPermissionResponses(permissions), nil
}

// tenantAllowed 判断目标资源租户是否允许当前上下文访问。
// ctxTenant=0 表示平台级视角（无租户上下文），放行所有资源。
func tenantAllowed(resourceTenant, ctxTenant uint64) bool {
	return ctxTenant == 0 || resourceTenant == 0 || resourceTenant == ctxTenant
}
