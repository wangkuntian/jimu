package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/modules/role/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestPermissionServiceCreateMapsDuplicateNameToConflict(t *testing.T) {
	service := NewPermissionService(&fakePermissionRepository{createErr: &mysqlerr.MySQLError{Number: 1062, Message: "Duplicate entry"}})

	_, err := service.Create(context.Background(), CreatePermissionRequest{Name: "user:read", Resource: "user", Action: "read"})
	if permissionAppCode(err) != apperrors.CodeConflict {
		t.Fatalf("code = %d, want %d", permissionAppCode(err), apperrors.CodeConflict)
	}
}

func TestPermissionServiceListPassesPagination(t *testing.T) {
	repo := &fakePermissionRepository{permissions: []domain.Permission{{ID: 1, Name: "user:read"}}, total: 12}
	service := NewPermissionService(repo)

	permissions, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 12 || len(permissions) != 1 || permissions[0].Name != "user:read" {
		t.Fatalf("permissions = %#v total = %d", permissions, total)
	}
}

func TestPermissionServiceUpdateMapsNotFound(t *testing.T) {
	service := NewPermissionService(&fakePermissionRepository{findErr: gorm.ErrRecordNotFound})

	err := service.Update(context.Background(), 8, UpdatePermissionRequest{Name: "user:read", Resource: "user", Action: "read"})
	if permissionAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", permissionAppCode(err), apperrors.CodeNotFound)
	}
}

func TestPermissionServiceDeleteWrapsRepositoryError(t *testing.T) {
	cause := stderrors.New("sql: connection refused")
	service := NewPermissionService(&fakePermissionRepository{deleteErr: cause})

	err := service.Delete(context.Background(), 8)
	if permissionAppCode(err) != apperrors.CodeInternalError || !stderrors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
}

type fakePermissionRepository struct {
	permission  *domain.Permission
	permissions []domain.Permission
	total       int64
	findErr     error
	createErr   error
	updateErr   error
	deleteErr   error
	offset      int
	limit       int
	sort        string
	order       string
}

func (r *fakePermissionRepository) FindByID(context.Context, uint64) (*domain.Permission, error) {
	if r.permission != nil {
		return r.permission, r.findErr
	}
	return &domain.Permission{}, r.findErr
}

func (r *fakePermissionRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.Permission, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.permissions, r.total, nil
}

func (r *fakePermissionRepository) Create(context.Context, *domain.Permission) error {
	return r.createErr
}
func (r *fakePermissionRepository) Update(context.Context, *domain.Permission) error {
	return r.updateErr
}
func (r *fakePermissionRepository) Delete(context.Context, uint64) error {
	return r.deleteErr
}

func permissionAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
