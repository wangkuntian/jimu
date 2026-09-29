package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/capabilities/tenant/domain"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type TenantService struct {
	repo domain.TenantRepository
}

func NewTenantService(repo domain.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

// Create 创建租户。租户编码全局唯一且不可修改，统一转小写存储。
func (s *TenantService) Create(ctx context.Context, req CreateTenantRequest) (*TenantResponse, error) {
	code := tenant.NormalizeCode(req.Code)
	if !tenant.ValidCode(code) {
		return nil, errors.New(errors.CodeTenantCodeFormat, "tenant code must match [a-zA-Z0-9_-] (1-64 chars)")
	}

	existing, err := s.repo.FindByCode(ctx, code)
	if err == nil && existing != nil {
		return nil, errors.New(errors.CodeTenantExists, "tenant code already exists")
	}
	if err != nil && !isNotFound(err) {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to find tenant by code", err)
	}

	t := &domain.Tenant{Code: code, Name: req.Name, Status: 1}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create tenant", err)
	}
	resp := ToTenantResponse(*t)
	return &resp, nil
}

// Get 按 ID 获取租户。
func (s *TenantService) Get(ctx context.Context, id uint64) (*TenantResponse, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.Wrap(errors.CodeTenantNotFound, "tenant not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get tenant", err)
	}
	resp := ToTenantResponse(*t)
	return &resp, nil
}

// List 分页获取租户列表。
func (s *TenantService) List(ctx context.Context, p pagination.Pagination) ([]TenantResponse, int64, error) {
	tenants, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list tenants", err)
	}
	return ToTenantResponses(tenants), total, nil
}

// Update 更新租户名称/状态。编码不可修改。
func (s *TenantService) Update(ctx context.Context, id uint64, req UpdateTenantRequest) error {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return errors.Wrap(errors.CodeTenantNotFound, "tenant not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get tenant", err)
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if err := s.repo.Update(ctx, t); err != nil {
		if stderrors.Is(err, dbutil.ErrConcurrentUpdate) {
			return errors.Wrap(errors.CodeConflict, "tenant was modified concurrently, please retry", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update tenant", err)
	}
	return nil
}

// Delete 删除租户（软删除）。默认租户受保护，不可删除。
func (s *TenantService) Delete(ctx context.Context, id uint64) error {
	if id == tenant.DefaultTenantID {
		return errors.New(errors.CodeTenantProtected, "default tenant cannot be deleted")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if isNotFound(err) {
			return errors.Wrap(errors.CodeTenantNotFound, "tenant not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get tenant", err)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete tenant", err)
	}
	return nil
}

func isNotFound(err error) bool {
	return err != nil && (err == gorm.ErrRecordNotFound || err.Error() == gorm.ErrRecordNotFound.Error())
}
