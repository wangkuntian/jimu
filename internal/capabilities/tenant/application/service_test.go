package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type fakeTenantRepository struct {
	tenant  *domain.Tenant
	tenants []domain.Tenant
	total   int64
	findErr error
	listErr error

	created *domain.Tenant
	updated *domain.Tenant
	deleted uint64
}

func (r *fakeTenantRepository) FindByID(context.Context, uint64) (*domain.Tenant, error) {
	return r.tenant, r.findErr
}

func (r *fakeTenantRepository) FindByCode(context.Context, string) (*domain.Tenant, error) {
	return r.tenant, r.findErr
}

func (r *fakeTenantRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.Tenant, int64, error) {
	return r.tenants, r.total, r.listErr
}

func (r *fakeTenantRepository) Create(_ context.Context, t *domain.Tenant) error {
	r.created = t
	return nil
}

func (r *fakeTenantRepository) Update(_ context.Context, t *domain.Tenant) error {
	r.updated = t
	return nil
}

func (r *fakeTenantRepository) Delete(_ context.Context, id uint64) error {
	r.deleted = id
	return nil
}

func tenantAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

func TestTenantServiceCreate(t *testing.T) {
	t.Run("成功创建并默认启用", func(t *testing.T) {
		repo := &fakeTenantRepository{}
		resp, err := NewTenantService(repo).Create(context.Background(), CreateTenantRequest{Code: "acme", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		if repo.created == nil || repo.created.Code != "acme" || repo.created.Status != 1 {
			t.Fatalf("created = %#v", repo.created)
		}
		if resp.Code != "acme" || resp.Status != 1 {
			t.Fatalf("resp = %#v", resp)
		}
	})

	t.Run("编码重复返回冲突", func(t *testing.T) {
		repo := &fakeTenantRepository{tenant: &domain.Tenant{ID: 9, Code: "acme"}}
		_, err := NewTenantService(repo).Create(context.Background(), CreateTenantRequest{Code: "acme", Name: "Acme"})
		if tenantAppCode(err) != apperrors.CodeTenantExists {
			t.Fatalf("code = %d, want %d", tenantAppCode(err), apperrors.CodeTenantExists)
		}
	})

	t.Run("编码统一转小写", func(t *testing.T) {
		repo := &fakeTenantRepository{}
		resp, err := NewTenantService(repo).Create(context.Background(), CreateTenantRequest{Code: "ACME", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		if repo.created == nil || repo.created.Code != "acme" {
			t.Fatalf("created code = %v, want acme", repo.created)
		}
		if resp.Code != "acme" {
			t.Fatalf("resp code = %q, want acme", resp.Code)
		}
	})

	t.Run("编码格式无效", func(t *testing.T) {
		for _, code := range []string{"has space", "中文", "a/b"} {
			_, err := NewTenantService(&fakeTenantRepository{}).Create(context.Background(), CreateTenantRequest{Code: code, Name: "x"})
			if tenantAppCode(err) != apperrors.CodeTenantCodeFormat {
				t.Fatalf("code %q err = %d, want %d", code, tenantAppCode(err), apperrors.CodeTenantCodeFormat)
			}
		}
	})
}

func TestTenantServiceGetNotFound(t *testing.T) {
	_, err := NewTenantService(&fakeTenantRepository{findErr: gorm.ErrRecordNotFound}).Get(context.Background(), 9)
	if tenantAppCode(err) != apperrors.CodeTenantNotFound {
		t.Fatalf("code = %d, want %d", tenantAppCode(err), apperrors.CodeTenantNotFound)
	}
}

func TestTenantServiceListPassesPagination(t *testing.T) {
	repo := &fakeTenantRepository{tenants: []domain.Tenant{{ID: 1, Code: "default", Name: "默认租户"}}, total: 1}
	service := NewTenantService(repo)

	tenants, total, err := service.List(context.Background(), pagination.Pagination{Page: 1, PageSize: 10, Sort: "code", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(tenants) != 1 || tenants[0].Code != "default" {
		t.Fatalf("tenants = %#v total = %d", tenants, total)
	}
}

func TestTenantServiceUpdate(t *testing.T) {
	t.Run("更新名称与状态", func(t *testing.T) {
		repo := &fakeTenantRepository{tenant: &domain.Tenant{ID: 2, Code: "acme", Name: "Acme", Status: 1}}
		name := "Acme Inc"
		status := int8(0)
		if err := NewTenantService(repo).Update(context.Background(), 2, UpdateTenantRequest{Name: &name, Status: &status}); err != nil {
			t.Fatal(err)
		}
		if repo.updated == nil || repo.updated.Name != "Acme Inc" || repo.updated.Status != 0 {
			t.Fatalf("updated = %#v", repo.updated)
		}
	})

	t.Run("租户不存在", func(t *testing.T) {
		err := NewTenantService(&fakeTenantRepository{findErr: gorm.ErrRecordNotFound}).Update(context.Background(), 9, UpdateTenantRequest{})
		if tenantAppCode(err) != apperrors.CodeTenantNotFound {
			t.Fatalf("code = %d, want %d", tenantAppCode(err), apperrors.CodeTenantNotFound)
		}
	})
}

func TestTenantServiceDeleteProtectsDefaultTenant(t *testing.T) {
	// 默认租户（id=1）不可删除
	err := NewTenantService(&fakeTenantRepository{}).Delete(context.Background(), tenant.DefaultTenantID)
	if tenantAppCode(err) != apperrors.CodeTenantProtected {
		t.Fatalf("code = %d, want %d", tenantAppCode(err), apperrors.CodeTenantProtected)
	}

	// 普通租户可删除
	repo := &fakeTenantRepository{tenant: &domain.Tenant{ID: 2, Code: "acme"}}
	if err := NewTenantService(repo).Delete(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if repo.deleted != 2 {
		t.Fatalf("deleted id = %d, want 2", repo.deleted)
	}

	// 不存在的租户
	err = NewTenantService(&fakeTenantRepository{findErr: gorm.ErrRecordNotFound}).Delete(context.Background(), 9)
	if tenantAppCode(err) != apperrors.CodeTenantNotFound {
		t.Fatalf("code = %d, want %d", tenantAppCode(err), apperrors.CodeTenantNotFound)
	}
}
