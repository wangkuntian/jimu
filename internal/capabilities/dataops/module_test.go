package dataops

import (
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDataopsModuleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestModuleNameAndContract(t *testing.T) {
	m := New(newDataopsModuleDB(t))
	assert.Equal(t, "dataops", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.Equal(t, "dataops", m.Descriptor().Name)
}

// TestModuleRegisterHTTP 用户导入端点注册且恰好一次（原 admin 归属）。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(newDataopsModuleDB(t)).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	assert.Equal(t, 1, got["POST /api/v1/admin/users/import/preview"])
	assert.Equal(t, 1, got["POST /api/v1/admin/users/import"])
	assert.Equal(t, 1, got["GET /api/v1/admin/users/import/template"])
	assert.Equal(t, 1, got["GET /api/v1/admin/users/import/:id"])
}
