package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/role/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type PermissionService struct {
	repo domain.PermissionRepository
}

func NewPermissionService(repo domain.PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) Create(ctx context.Context, req CreatePermissionRequest) (*PermissionResponse, error) {
	perm := &domain.Permission{Name: req.Name, Resource: req.Resource, Action: req.Action}
	if err := s.repo.Create(ctx, perm); err != nil {
		if isDuplicateKey(err) {
			return nil, errors.Wrap(errors.CodeConflict, "permission already exists", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create permission", err)
	}
	resp := ToPermissionResponse(*perm)
	return &resp, nil
}

func (s *PermissionService) Get(ctx context.Context, id uint64) (*PermissionResponse, error) {
	perm, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "permission not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get permission", err)
	}
	resp := ToPermissionResponse(*perm)
	return &resp, nil
}

func (s *PermissionService) List(ctx context.Context, p pagination.Pagination) ([]PermissionResponse, int64, error) {
	permissions, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list permissions", err)
	}
	return ToPermissionResponses(permissions), total, nil
}

func (s *PermissionService) Update(ctx context.Context, id uint64, req UpdatePermissionRequest) error {
	perm, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "permission not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get permission", err)
	}
	perm.Name = req.Name
	perm.Resource = req.Resource
	perm.Action = req.Action
	if err := s.repo.Update(ctx, perm); err != nil {
		if isDuplicateKey(err) {
			return errors.Wrap(errors.CodeConflict, "permission already exists", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update permission", err)
	}
	return nil
}

func (s *PermissionService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete permission", err)
	}
	return nil
}
