package feature

import (
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestModuleNameAndContract 能力名与契约实现。
func TestModuleNameAndContract(t *testing.T) {
	m := New(nil)
	assert.Equal(t, "feature", m.Name())
	var _ contract.Module = m
	m.RegisterJobs(nil)
	m.RegisterEvents(nil)
	assert.NotNil(t, m.Manager())
}

// TestModuleDescriptor 描述符形态。
func TestModuleDescriptor(t *testing.T) {
	d := New(nil).Descriptor()
	assert.Equal(t, "feature", d.Name)
	assert.Equal(t, contract.MountProtected, d.Normalized())
}

// TestModuleRegisterHTTP Feature Flag 管理端点上架且恰好一次。
func TestModuleRegisterHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(nil).RegisterHTTP(r)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	assert.Equal(t, 1, got["GET /api/v1/admin/features"])
	assert.Equal(t, 1, got["PUT /api/v1/admin/features/:name"])
}
