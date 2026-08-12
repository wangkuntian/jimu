package admin

import (
	"testing"

	adminapp "jimu/internal/modules/admin/application"

	"github.com/gin-gonic/gin"
)

// fakeRouter 捕获注册的组路径
type fakeRouter struct {
	groupPaths []string
	*gin.Engine
}

func newFakeRouter() *fakeRouter {
	gin.SetMode(gin.TestMode)
	return &fakeRouter{Engine: gin.New()}
}

func (f *fakeRouter) Group(relativePath string, _ ...gin.HandlerFunc) *gin.RouterGroup {
	f.groupPaths = append(f.groupPaths, relativePath)
	return f.Engine.Group(relativePath)
}

// TestAdminRoutesUseAPIV1Prefix 验证管理端路由统一挂在 /api/v1/admin 前缀下
func TestAdminRoutesUseAPIV1Prefix(t *testing.T) {
	m := &Module{service: adminapp.NewService("test", "test", nil)}
	r := newFakeRouter()
	m.RegisterHTTP(r)

	found := false
	for _, p := range r.groupPaths {
		if p == "/api/v1/admin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("admin module 未挂载 /api/v1/admin 前缀，实际: %v", r.groupPaths)
	}
}
