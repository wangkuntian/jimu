package queue

import (
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newQueueModuleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestModuleNameAndContract(t *testing.T) {
	m := NewModule(newQueueModuleDB(t), nil)
	assert.Equal(t, "queue", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.Equal(t, "queue", m.Descriptor().Name)
}

// TestModuleRegisterHTTP 任务与作业管理端点全部注册且恰好一次（原 admin 归属）。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewModule(newQueueModuleDB(t), nil).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"GET /api/v1/admin/jobs",
		"POST /api/v1/admin/jobs",
		"GET /api/v1/admin/jobs/:id",
		"POST /api/v1/admin/jobs/:id/retry",
		"GET /api/v1/admin/jobs/dead-letters",
		"POST /api/v1/admin/jobs/dead-letters/:id/resolve",
		"GET /api/v1/admin/tasks",
		"POST /api/v1/admin/tasks/:id/run",
		"POST /api/v1/admin/tasks/:id/toggle",
		"GET /api/v1/admin/tasks/:id/history",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}

// TestNewModuleNilScheduler 无调度器时任务端点仍注册（调用返回错误）。
func TestNewModuleNilScheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewModule(newQueueModuleDB(t), nil)
	assert.Nil(t, m.sched)
	r := gin.New()
	m.RegisterHTTP(r)
	assert.NotEmpty(t, r.Routes())
}
