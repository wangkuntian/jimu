package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/role/domain"
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
	resp := ToRoleResponse(*role)
	return &resp, nil
}

func (s *RoleService) List(ctx context.Context, p pagination.Pagination) ([]RoleResponse, int64, error) {
	roles, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
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
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete role", err)
	}
	return nil
}

func (s *RoleService) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return s.repo.AssignPermissions(ctx, roleID, permissionIDs)
}

func (s *RoleService) GetPermissions(ctx context.Context, roleID uint64) ([]PermissionResponse, error) {
	permissions, err := s.repo.GetPermissions(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get permissions", err)
	}
	return ToPermissionResponses(permissions), nil
}
