package application

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"jimu/internal/modules/user/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

func TestUserResponseDoesNotContainPassword(t *testing.T) {
	got := ToUserResponse(domain.User{ID: 1, Username: "alice", Password: "hash", Status: 1})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte("hash")) || bytes.Contains(b, []byte("password")) || bytes.Contains(b, []byte("deleted")) {
		t.Fatalf("sensitive response: %s", b)
	}
}

func TestUserServiceListPassesPaginationAndReturnsDTO(t *testing.T) {
	repo := &fakeUserRepository{
		users: []domain.User{{ID: 7, Username: "alice", Password: "hash", Status: 1}},
		total: 33,
	}
	service := NewUserService(repo, nil)

	users, total, err := service.List(context.Background(), pagination.Pagination{Page: 3, PageSize: 10, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 20 || repo.limit != 10 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 33 || len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("users = %#v total = %d", users, total)
	}
}

func TestUserServiceGetMapsNotFound(t *testing.T) {
	service := NewUserService(&fakeUserRepository{findErr: gorm.ErrRecordNotFound}, nil)

	_, err := service.Get(context.Background(), 9)
	if appCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeNotFound)
	}
}

type fakeUserRepository struct {
	user    *domain.User
	users   []domain.User
	total   int64
	findErr error
	listErr error
	offset  int
	limit   int
	sort    string
	order   string
}

func (r *fakeUserRepository) FindByID(context.Context, uint64) (*domain.User, error) {
	return r.user, r.findErr
}

func (r *fakeUserRepository) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, stderrors.New("not found")
}

func (r *fakeUserRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.User, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.users, r.total, r.listErr
}

func (r *fakeUserRepository) Create(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepository) Update(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepository) Delete(context.Context, uint64) error       { return nil }

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
