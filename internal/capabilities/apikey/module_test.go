package apikey

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

func newAPIKeyModuleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestModuleNameAndContract(t *testing.T) {
	m := New(newAPIKeyModuleDB(t))
	assert.Equal(t, "apikey", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.Equal(t, "apikey", m.Descriptor().Name)
}

// TestModuleDescriptor 描述符：软依赖 tenant（可选配额），自有 api_keys 表。
func TestModuleDescriptor(t *testing.T) {
	d := New(newAPIKeyModuleDB(t)).Descriptor()
	assert.Equal(t, []string{"tenant"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"api_keys"}, d.Owns)
}

// TestModuleRegisterHTTP API Key 管理端点注册且恰好一次（原 admin 归属）。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(newAPIKeyModuleDB(t)).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	assert.Equal(t, 1, got["GET /api/v1/admin/apikeys"])
	assert.Equal(t, 1, got["POST /api/v1/admin/apikeys"])
	assert.Equal(t, 1, got["GET /api/v1/admin/apikeys/:id"])
	assert.Equal(t, 1, got["DELETE /api/v1/admin/apikeys/:id"])
}

// TestModuleWithQuota 注入配额校验不 panic。
func TestModuleWithQuota(t *testing.T) {
	m := New(newAPIKeyModuleDB(t), fakeAPIKeyQuota{})
	assert.Equal(t, "apikey", m.Name())
}

type fakeAPIKeyQuota struct{}

func (fakeAPIKeyQuota) CheckAPIKeyQuota(context.Context, uint64) error { return nil }
