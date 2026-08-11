package application

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/modules/user/domain"
	"jimu/internal/platform/outbox"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
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

// recordingOutboxStore 记录 Add 的事件，其余方法无操作
type recordingOutboxStore struct {
	events []outbox.Event
}

func (o *recordingOutboxStore) Add(_ context.Context, _ interface{}, events ...outbox.Event) error {
	o.events = append(o.events, events...)
	return nil
}
func (o *recordingOutboxStore) FetchUnpublish(context.Context, int) ([]outbox.Event, error) {
	return nil, nil
}
func (o *recordingOutboxStore) MarkPublished(context.Context, []uint64) error   { return nil }
func (o *recordingOutboxStore) MarkFailed(context.Context, uint64, error) error { return nil }

// createOutboxUserService 构造带 recording outbox 的 UserService
func createOutboxUserService() (*UserService, *recordingOutboxStore) {
	store := &recordingOutboxStore{}
	ob := outbox.New(store, nil)
	svc := NewUserService(&fakeOutboxUserRepo{}, nil, ob)
	return svc, store
}

type fakeOutboxUserRepo struct{}

func (r *fakeOutboxUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice"}, nil
}
func (r *fakeOutboxUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeOutboxUserRepo) List(context.Context, int, int, string, string) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeOutboxUserRepo) Create(_ context.Context, u *domain.User) error {
	u.ID = 1
	return nil
}
func (r *fakeOutboxUserRepo) Update(context.Context, *domain.User) error { return nil }
func (r *fakeOutboxUserRepo) Delete(context.Context, uint64) error       { return nil }

func TestCreateWritesOutbox(t *testing.T) {
	svc, store := createOutboxUserService()
	_, err := svc.Create(context.Background(), CreateUserRequest{Username: "alice", Password: "password123"})
	assert.NoError(t, err)
	assert.Len(t, store.events, 1)
	assert.Equal(t, contract.EventUserCreated, store.events[0].EventType)
	assert.Equal(t, "user:1", store.events[0].AggregateID)
}

func TestUpdateAndDeleteWriteOutbox(t *testing.T) {
	svc, store := createOutboxUserService()

	status := int8(0)
	assert.NoError(t, svc.Update(context.Background(), 1, UpdateUserRequest{Status: &status}))
	assert.Len(t, store.events, 1)
	assert.Equal(t, contract.EventUserUpdated, store.events[0].EventType)

	assert.NoError(t, svc.Delete(context.Background(), 1))
	assert.Len(t, store.events, 2)
	assert.Equal(t, contract.EventUserDeleted, store.events[1].EventType)
}
