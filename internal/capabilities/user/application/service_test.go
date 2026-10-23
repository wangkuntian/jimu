package application

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	users, total, err := service.List(context.Background(), pagination.Pagination{Page: 3, PageSize: 10, Sort: "created_at", Order: "asc"}, domain.UserListFilter{})
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

func TestUserServiceBatchDelete(t *testing.T) {
	ctx := context.Background()

	// 全部失败：每个 id 找不到用户
	_, err := NewUserService(&fakeUserRepository{
		user:    &domain.User{ID: 1},
		findErr: gorm.ErrRecordNotFound,
	}, nil).BatchDelete(ctx, []uint64{1, 2})
	if appCode(err) != apperrors.CodeInternalError {
		t.Fatalf("all-fail code = %d, want %d", appCode(err), apperrors.CodeInternalError)
	}

	// 全部失败：找到但删除报错
	_, err = NewUserService(&fakeUserRepository{
		user:      &domain.User{ID: 1},
		deleteErr: stderrors.New("boom"),
	}, nil).BatchDelete(ctx, []uint64{1})
	if appCode(err) != apperrors.CodeInternalError {
		t.Fatalf("delete-fail code = %d, want %d", appCode(err), apperrors.CodeInternalError)
	}

	// 全部成功
	res, err := NewUserService(&fakeUserRepository{user: &domain.User{ID: 1}}, nil).BatchDelete(ctx, []uint64{1})
	if err != nil || res.Success != 1 || res.Failed != 0 {
		t.Fatalf("success res = %#v err = %v", res, err)
	}
}

func TestUserServiceGetMapsNotFound(t *testing.T) {
	service := NewUserService(&fakeUserRepository{findErr: gorm.ErrRecordNotFound}, nil)

	_, err := service.Get(context.Background(), 9)
	if appCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeNotFound)
	}
}

func TestUserServiceCreateAssignsTenant(t *testing.T) {
	// 上下文带租户：新用户归属该租户
	repo := &fakeUserRepository{}
	svc := NewUserService(repo, nil)
	if _, err := svc.Create(tenant.WithTenant(context.Background(), 7), CreateUserRequest{Username: "alice", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	if repo.created == nil || repo.created.TenantID != 7 {
		t.Fatalf("created tenant = %v, want 7", repo.created)
	}

	// 上下文无租户：平台管理员必须显式指定目标租户
	repo2 := &fakeUserRepository{}
	tenants := fakeTenantInfoSource{infos: map[uint64]contract.TenantInfo{9: {ID: 9, Name: "Acme"}}}
	if _, err := NewUserService(repo2, tenants).Create(context.Background(), CreateUserRequest{TenantID: "9", Username: "bob", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	if repo2.created == nil || repo2.created.TenantID != 9 {
		t.Fatalf("target tenant = %v, want 9", repo2.created)
	}
}

func TestUserServiceCreateRequiresTenantWithoutContext(t *testing.T) {
	repo := &fakeUserRepository{}
	_, err := NewUserService(repo).Create(context.Background(), CreateUserRequest{Username: "bob", Password: "password1"})
	assert.Equal(t, apperrors.CodeInvalidParam, appCode(err))
	assert.Nil(t, repo.created)
}

func TestUserServiceGetRejectsCrossTenant(t *testing.T) {
	service := NewUserService(&fakeUserRepository{user: &domain.User{ID: 9, TenantID: 2}}, nil)
	ctx := tenant.WithTenant(context.Background(), 1)

	_, err := service.Get(ctx, 9)
	if appCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeNotFound)
	}
}

func TestUserServiceListPassesTenantToRepo(t *testing.T) {
	repo := &fakeUserRepository{users: []domain.User{{ID: 7, Username: "alice"}}, total: 1}
	service := NewUserService(repo, nil)
	ctx := tenant.WithTenant(context.Background(), 7)

	if _, _, err := service.List(ctx, pagination.Pagination{Page: 1, PageSize: 10, Sort: "id", Order: "desc"}, domain.UserListFilter{}); err != nil {
		t.Fatal(err)
	}
	if repo.filter.TenantID != 7 {
		t.Fatalf("repo tenantID = %d, want 7", repo.filter.TenantID)
	}
}

type fakeUserRepository struct {
	user      *domain.User
	users     []domain.User
	total     int64
	findErr   error
	listErr   error
	deleteErr error
	created   *domain.User
	updated   *domain.User
	offset    int
	limit     int
	sort      string
	order     string
	filter    domain.UserListFilter
	create    func(ctx context.Context, user *domain.User) error
}

func (r *fakeUserRepository) FindByID(context.Context, uint64) (*domain.User, error) {
	return r.user, r.findErr
}

func (r *fakeUserRepository) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepository) List(_ context.Context, filter domain.UserListFilter, offset, limit int, sort, order string) ([]domain.User, int64, error) {
	r.filter = filter
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.users, r.total, r.listErr
}

func (r *fakeUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.created = user
	if r.create != nil {
		return r.create(ctx, user)
	}
	return nil
}
func (r *fakeUserRepository) Update(_ context.Context, user *domain.User) error {
	r.updated = user
	return nil
}
func (r *fakeUserRepository) Delete(context.Context, uint64) error { return r.deleteErr }
func (r *fakeUserRepository) FindByEmailHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepository) FindByPhoneHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepository) UpdatePassword(context.Context, uint64, string) error { return nil }

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

type recordingEventWriter struct{ events []contract.OutboxEvent }

func (o *recordingEventWriter) WriteEvent(_ context.Context, _ any, event contract.OutboxEvent) error {
	o.events = append(o.events, event)
	return nil
}

// createOutboxUserService 构造带 recording outbox 的 UserService
func createOutboxUserService() (*UserService, *recordingEventWriter) {
	store := &recordingEventWriter{}
	svc := NewUserService(&fakeOutboxUserRepo{}, store, fakeTenantInfoSource{
		infos: map[uint64]contract.TenantInfo{1: {ID: 1, Name: "默认租户"}},
	})
	return svc, store
}

type fakeOutboxUserRepo struct{}

func (r *fakeOutboxUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice", Status: 1}, nil
}
func (r *fakeOutboxUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeOutboxUserRepo) List(context.Context, domain.UserListFilter, int, int, string, string) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeOutboxUserRepo) Create(_ context.Context, u *domain.User) error {
	u.ID = 1
	return nil
}
func (r *fakeOutboxUserRepo) Update(context.Context, *domain.User) error { return nil }
func (r *fakeOutboxUserRepo) Delete(context.Context, uint64) error       { return nil }
func (r *fakeOutboxUserRepo) FindByEmailHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeOutboxUserRepo) FindByPhoneHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeOutboxUserRepo) UpdatePassword(context.Context, uint64, string) error { return nil }

func TestCreateWritesOutbox(t *testing.T) {
	svc, store := createOutboxUserService()
	_, err := svc.Create(context.Background(), CreateUserRequest{TenantID: "1", Username: "alice", Password: "password123"})
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

type fakeTenantInfoSource struct {
	infos map[uint64]contract.TenantInfo
	err   error
}

func (f fakeTenantInfoSource) GetByIDs(context.Context, []uint64) (map[uint64]contract.TenantInfo, error) {
	return f.infos, f.err
}

type fakeUserRoleManager struct {
	assigned  map[uint64][]uint64
	roles     map[uint64][]contract.UserRole
	assignErr error
	rolesErr  error
}

func (f *fakeUserRoleManager) AssignRoleIDs(_ context.Context, userID uint64, roleIDs []uint64) error {
	if f.assigned == nil {
		f.assigned = make(map[uint64][]uint64)
	}
	f.assigned[userID] = roleIDs
	return f.assignErr
}

func (f *fakeUserRoleManager) RolesForUsers(context.Context, []uint64) (map[uint64][]contract.UserRole, error) {
	return f.roles, f.rolesErr
}

func TestUserServiceCreateAssignsRoles(t *testing.T) {
	repo := &fakeUserRepository{create: func(_ context.Context, user *domain.User) error {
		user.ID = 7
		return nil
	}}
	roles := &fakeUserRoleManager{
		roles: map[uint64][]contract.UserRole{7: {{ID: 10, Name: "租户管理员"}}},
	}
	tenants := fakeTenantInfoSource{infos: map[uint64]contract.TenantInfo{2: {ID: 2, Name: "Acme"}}}
	svc := NewUserService(repo, roles, tenants)

	got, err := svc.Create(context.Background(), CreateUserRequest{
		TenantID: "2",
		Username: "alice",
		Password: "password1",
		RoleIDs:  []string{"10"},
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), repo.created.TenantID)
	assert.Equal(t, []uint64{10}, roles.assigned[7])
	assert.Equal(t, "7", got.ID)
	assert.Equal(t, applicationTenantRef("2", "Acme"), got.Tenant)
	assert.Equal(t, []RoleRef{{ID: "10", Name: "租户管理员"}}, got.Roles)
}

func TestUserServiceUpdateContactFields(t *testing.T) {
	repo := &fakeUserRepository{user: &domain.User{ID: 7, TenantID: 1, Status: 1}}
	svc := NewUserService(repo)
	email := "alice@example.com"
	phone := "13800138000"

	require.NoError(t, svc.Update(tenant.WithTenant(context.Background(), 1), 7, UpdateUserRequest{
		Email: &email,
		Phone: &phone,
	}))
	require.NotNil(t, repo.updated)
	assert.Equal(t, email, repo.updated.Email)
	assert.Equal(t, phone, repo.updated.Phone)
}

func TestUserServiceListHydratesTenantAndRoles(t *testing.T) {
	repo := &fakeUserRepository{
		users: []domain.User{{ID: 7, Username: "alice", TenantID: 2, Status: 1}},
		total: 1,
	}
	roles := &fakeUserRoleManager{roles: map[uint64][]contract.UserRole{7: {{ID: 10, Name: "租户管理员"}}}}
	tenants := fakeTenantInfoSource{infos: map[uint64]contract.TenantInfo{2: {ID: 2, Name: "Acme"}}}
	svc := NewUserService(repo, roles, tenants)

	users, total, err := svc.List(context.Background(), pagination.Pagination{Page: 1, PageSize: 20, Sort: "id", Order: "desc"}, domain.UserListFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	assert.Equal(t, "7", users[0].ID)
	assert.Equal(t, TenantRef{ID: "2", Name: "Acme"}, users[0].Tenant)
	assert.Equal(t, []RoleRef{{ID: "10", Name: "租户管理员"}}, users[0].Roles)
}

func TestUserServiceCreateRejectsCrossTenant(t *testing.T) {
	svc := NewUserService(&fakeUserRepository{})
	_, err := svc.Create(tenant.WithTenant(context.Background(), 1), CreateUserRequest{
		TenantID: "2",
		Username: "alice",
		Password: "password1",
	})
	assert.Equal(t, apperrors.CodeForbidden, appCode(err))
}

func TestUserServicePlatformScopeUsesRequestedTenant(t *testing.T) {
	repo := &fakeUserRepository{}
	roles := &fakeUserRoleManager{}
	tenants := fakeTenantInfoSource{infos: map[uint64]contract.TenantInfo{2: {ID: 2, Name: "Acme"}}}
	svc := NewUserService(repo, roles, tenants)
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), tenant.DefaultTenantID))

	got, err := svc.Create(ctx, CreateUserRequest{
		TenantID: "2",
		Username: "alice",
		Password: "password1",
		RoleIDs:  []string{"10"},
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), repo.created.TenantID)
	assert.Equal(t, "2", got.Tenant.ID)
}

func TestUserServiceTenantAdminScopeStillUsesContextTenant(t *testing.T) {
	repo := &fakeUserRepository{users: []domain.User{{ID: 7, Username: "alice", TenantID: 7}}, total: 1}
	svc := NewUserService(repo)
	ctx := tenant.WithTenant(context.Background(), 7)

	_, _, err := svc.List(ctx, pagination.Pagination{Page: 1, PageSize: 20}, domain.UserListFilter{TenantID: 8})
	assert.Equal(t, apperrors.CodeForbidden, appCode(err))

	_, _, err = svc.List(ctx, pagination.Pagination{Page: 1, PageSize: 20}, domain.UserListFilter{})
	require.NoError(t, err)
	assert.Equal(t, uint64(7), repo.filter.TenantID)
}

func applicationTenantRef(id, name string) TenantRef {
	return TenantRef{ID: id, Name: name}
}
