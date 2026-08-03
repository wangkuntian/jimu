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

func TestRoleServiceCreateMapsDuplicateNameToConflict(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{createErr: &mysqlerr.MySQLError{Number: 1062, Message: "Duplicate entry"}})

	_, err := service.Create(context.Background(), CreateRoleRequest{Name: "admin", Description: "ops"})
	if roleAppCode(err) != apperrors.CodeConflict {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeConflict)
	}
}

func TestRoleServiceListPassesPagination(t *testing.T) {
	repo := &fakeRoleRepository{roles: []domain.Role{{ID: 1, Name: "admin"}}, total: 12}
	service := NewRoleService(repo)

	roles, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 12 || len(roles) != 1 || roles[0].Name != "admin" {
		t.Fatalf("roles = %#v total = %d", roles, total)
	}
}

func TestRoleServiceUpdateMapsNotFound(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{findErr: gorm.ErrRecordNotFound})

	err := service.Update(context.Background(), 8, UpdateRoleRequest{Name: "admin"})
	if roleAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeNotFound)
	}
}

func TestRoleServiceDeleteWrapsRepositoryError(t *testing.T) {
	cause := stderrors.New("sql: connection refused")
	service := NewRoleService(&fakeRoleRepository{deleteErr: cause})

	err := service.Delete(context.Background(), 8)
	if roleAppCode(err) != apperrors.CodeInternalError || !stderrors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
}

type fakeRoleRepository struct {
	role      *domain.Role
	roles     []domain.Role
	total     int64
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	offset    int
	limit     int
	sort      string
	order     string
}

func (r *fakeRoleRepository) FindByID(context.Context, uint64) (*domain.Role, error) {
	if r.role != nil {
		return r.role, r.findErr
	}
	return &domain.Role{}, r.findErr
}

func (r *fakeRoleRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.Role, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.roles, r.total, nil
}

func (r *fakeRoleRepository) Create(context.Context, *domain.Role) error { return r.createErr }
func (r *fakeRoleRepository) Update(context.Context, *domain.Role) error { return r.updateErr }
func (r *fakeRoleRepository) Delete(context.Context, uint64) error       { return r.deleteErr }
func (r *fakeRoleRepository) AssignPermissions(context.Context, uint64, []uint64) error {
	return nil
}
func (r *fakeRoleRepository) GetPermissions(context.Context, uint64) ([]domain.Permission, error) {
	return nil, nil
}

func roleAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
