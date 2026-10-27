package user

import (
	"context"
	"testing"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserModule(t *testing.T) *Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return New(db)
}

func TestModuleNameAndContract(t *testing.T) {
	m := newUserModule(t)
	assert.Equal(t, "user", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	assert.Equal(t, "user", m.Descriptor().Name)
}

// TestModuleDescriptor 描述符：软依赖 access/tenant（可选注入），自有 users 表。
func TestModuleDescriptor(t *testing.T) {
	d := newUserModule(t).Descriptor()
	assert.Equal(t, []string{"access", "tenant"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"users"}, d.Owns)
}

// TestModuleRegisterHTTP 平台/租户管理员共用的用户资源路由（角色分配归 access）。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	newUserModule(t).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /api/v1/users", "GET /api/v1/users", "GET /api/v1/users/:id",
		"PUT /api/v1/users/:id", "DELETE /api/v1/users/:id",
		"POST /api/v1/users/batch-delete", "GET /api/v1/users/export.csv",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
	assert.Zero(t, got["GET /api/v1/admin/users"], "用户管理不得再挂在管理端命名空间")
}

func TestModulePortInjection(t *testing.T) {
	m := newUserModule(t).WithRoles(fakeAssigner{}).WithQuota(fakeQuota{})
	assert.Equal(t, "user", m.Name())
}

func TestUserinfoSource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}))
	repo := newFakeUserRepo()
	src := NewUserinfoSource(repo)

	info, err := src.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(9), info.TenantID, "应带出归属租户")

	byName, err := src.FindByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), byName.ID)
}

func TestUserinfoSourceUserActive(t *testing.T) {
	repo := newFakeUserRepo()
	src := NewUserinfoSource(repo)

	active, err := src.(contract.UserStateSource).UserActive(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, active)

	repo.user.Status = 0
	active, err = src.(contract.UserStateSource).UserActive(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, active)

	repo.err = gorm.ErrRecordNotFound
	active, err = src.(contract.UserStateSource).UserActive(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, active)
}

type fakeUserRepo struct {
	user *domain.User
	err  error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{user: &domain.User{ID: 1, Username: "alice", TenantID: 9, Status: 1}}
}

func (r *fakeUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}
func (r *fakeUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice", TenantID: 9, Status: 1}, nil
}
func (r *fakeUserRepo) List(context.Context, domain.UserListFilter, int, int, string, string) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepo) Update(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepo) Delete(context.Context, uint64) error       { return nil }
func (r *fakeUserRepo) FindByEmailHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepo) FindByPhoneHash(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepo) UpdatePassword(context.Context, uint64, string) error { return nil }

type fakeAssigner struct{}

func (fakeAssigner) AssignRoleIDs(context.Context, uint64, []uint64) error { return nil }

func (fakeAssigner) RolesForUsers(context.Context, []uint64) (map[uint64][]contract.UserRole, error) {
	return nil, nil
}

func (fakeAssigner) EnsureCanDeactivateUser(context.Context, uint64) error { return nil }

func (fakeAssigner) RemoveUserRoles(context.Context, uint64) error { return nil }

type fakeQuota struct{}

func (fakeQuota) CheckUserQuota(context.Context, uint64) error { return nil }

var _ contract.UserRoleAssigner = fakeAssigner{}
