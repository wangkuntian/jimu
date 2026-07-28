package application

import (
	"context"

	"jimu/internal/modules/role/domain"
	"jimu/internal/shared/errors"
)

type PermissionService struct {
	repo domain.PermissionRepository
}

func NewPermissionService(repo domain.PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) Create(ctx context.Context, name, resource, action string) (*domain.Permission, error) {
	perm := &domain.Permission{Name: name, Resource: resource, Action: action}
	if err := s.repo.Create(ctx, perm); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create permission", err)
	}
	return perm, nil
}

func (s *PermissionService) Get(ctx context.Context, id uint64) (*domain.Permission, error) {
	perm, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeNotFound, "permission not found")
	}
	return perm, nil
}

func (s *PermissionService) List(ctx context.Context) ([]domain.Permission, error) {
	return s.repo.FindAll(ctx)
}

func (s *PermissionService) Update(ctx context.Context, id uint64, name, resource, action string) error {
	perm, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.CodeNotFound, "permission not found")
	}
	perm.Name = name
	perm.Resource = resource
	perm.Action = action
	return s.repo.Update(ctx, perm)
}

func (s *PermissionService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}
