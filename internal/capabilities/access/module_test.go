package access

import (
	"context"
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAccessTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// TestModuleNameAndContract 能力名与契约实现。
func TestModuleNameAndContract(t *testing.T) {
	m := New(newAccessTestDB(t))
	assert.Equal(t, "access", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.NotNil(t, m.UserRoleAssigner(), "应暴露 contract.UserRoleAssigner")
}

// TestModuleDescriptor 描述符：依赖 user，声明角色/权限/用户角色分配权限点。
func TestModuleDescriptor(t *testing.T) {
	m := New(newAccessTestDB(t))
	d := m.Descriptor()
	assert.Equal(t, "access", d.Name)
	assert.Equal(t, contract.MountProtected, d.Normalized())
	assert.ElementsMatch(t, []string{"user"}, d.Requires)
	// 角色 6 + 权限 5
	require.Len(t, d.Permissions, 11)
	assert.NotNil(t, d.Migrations)
}

// TestModuleRegisterHTTP 角色/权限路由 + 管理端「用户分配角色」端点。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New(newAccessTestDB(t))
	r := gin.New()
	m.RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /api/v1/roles",
		"GET /api/v1/roles",
		"GET /api/v1/roles/:id",
		"PUT /api/v1/roles/:id",
		"DELETE /api/v1/roles/:id",
		"POST /api/v1/roles/:id/permissions",
		"POST /api/v1/permissions",
		"GET /api/v1/permissions",
		"GET /api/v1/permissions/:id",
		"PUT /api/v1/permissions/:id",
		"DELETE /api/v1/permissions/:id",
		"POST /api/v1/admin/users/:id/roles",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}

// TestModuleWithQuota 注入配额校验不 panic（deps 为可选）。
func TestModuleWithQuota(t *testing.T) {
	m := New(newAccessTestDB(t), fakeQuota{})
	assert.Equal(t, "access", m.Name())
}

type fakeQuota struct{}

func (fakeQuota) CheckRoleQuota(_ context.Context, _ uint64) error { return nil }
