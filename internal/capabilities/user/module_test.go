package user

import (
	"context"
	"testing"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/domain"
	"jimu/internal/config"
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
	return New(db, config.Config{})
}

func TestModuleNameAndContract(t *testing.T) {
	m := newUserModule(t)
	assert.Equal(t, "user", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	assert.Equal(t, "user", m.Descriptor().Name)
}

// TestModuleRegisterHTTP 自助面 + 管理面用户路由（角色分配归 access，不得重复）。
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
		"GET /api/v1/admin/users", "POST /api/v1/admin/users",
		"GET /api/v1/admin/users/:id", "PUT /api/v1/admin/users/:id",
		"DELETE /api/v1/admin/users/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
	assert.Zero(t, got["POST /api/v1/admin/users/:id/roles"], "用户角色分配归 access")
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

type fakeUserRepo struct{}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{} }

func (r *fakeUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice", TenantID: 9, Status: 1}, nil
}
func (r *fakeUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice", TenantID: 9, Status: 1}, nil
}
func (r *fakeUserRepo) List(context.Context, uint64, int, int, string, string) ([]domain.User, int64, error) {
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

func (fakeAssigner) AssignRoles(context.Context, uint64, []string) error { return nil }

type fakeQuota struct{}

func (fakeQuota) CheckUserQuota(context.Context, uint64) error { return nil }

var _ application.UserRoleAssigner = fakeAssigner{}
