package application

import (
	"context"

	"jimu/internal/modules/role/domain"
	"jimu/internal/shared/errors"
)

type RoleService struct {
	repo domain.RoleRepository
}

func NewRoleService(repo domain.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Create(ctx context.Context, name, description string) (*domain.Role, error) {
	role := &domain.Role{Name: name, Description: description, Status: 1}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create role", err)
	}
	return role, nil
}

func (s *RoleService) Get(ctx context.Context, id uint64) (*domain.Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeRoleNotFound, "role not found")
	}
	return role, nil
}

func (s *RoleService) List(ctx context.Context) ([]domain.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *RoleService) Update(ctx context.Context, id uint64, name, description string) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.CodeRoleNotFound, "role not found")
	}
	role.Name = name
	role.Description = description
	return s.repo.Update(ctx, role)
}

func (s *RoleService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func (s *RoleService) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return s.repo.AssignPermissions(ctx, roleID, permissionIDs)
}

func (s *RoleService) GetPermissions(ctx context.Context, roleID uint64) ([]domain.Permission, error) {
	return s.repo.GetPermissions(ctx, roleID)
}
