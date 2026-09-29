package uploadsec

import (
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestModuleNameAndContract(t *testing.T) {
	m := New(newFakeStorage(), nil)
	assert.Equal(t, "uploadsec", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.Equal(t, "uploadsec", m.Descriptor().Name)
}

// TestModuleRegisterHTTPWithoutStorage storage 为空时不注册端点（与拆分前 admin 行为一致）。
func TestModuleRegisterHTTPWithoutStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(nil, nil).RegisterHTTP(r)
	assert.Empty(t, r.Routes())
}

// TestModuleRegisterHTTP 注入存储后注册 /admin/files 上传与删除端点。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(newFakeStorage(), nil).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	assert.Equal(t, 1, got["POST /api/v1/admin/files"])
	assert.Equal(t, 1, got["DELETE /api/v1/admin/files"])
}
