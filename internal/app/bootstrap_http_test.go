package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
)

// probeModule 记录是否被注册，并按声明的挂载点注册一个探针路由。
type probeModule struct {
	name       string
	desc       contract.Descriptor
	registered bool
}

func (p *probeModule) Name() string { return p.name }

func (p *probeModule) Descriptor() contract.Descriptor { return p.desc }

func (p *probeModule) RegisterHTTP(r contract.Router) {
	p.registered = true
	r.Group("/api/v1").GET("/"+p.name+"/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}

func (p *probeModule) RegisterJobs(contract.JobRegistry) {}

func (p *probeModule) RegisterEvents(contract.EventBus) {}

// protectedProvider 声明受保护中间件：命中时写入标记头。
type protectedProvider struct{ probeModule }

func (p *protectedProvider) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	return []gin.HandlerFunc{func(c *gin.Context) {
		c.Header("X-Protected", "1")
		c.Next()
	}}, nil
}

func TestRegisterHTTPAppliesProtectedMiddlewareByMountPoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 名称刻意不用 auth/oauth：当前实现按名称特判，本用例必须能在该实现下失败。
	self := &protectedProvider{probeModule{name: "probe-self", desc: contract.Descriptor{Name: "probe-self", Mount: contract.MountSelfManaged}}}
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}
	pub := &probeModule{name: "probe-pub", desc: contract.Descriptor{Name: "probe-pub", Mount: contract.MountPublic}}

	if err := registerHTTP(router, nil, nil, self, prot, pub); err != nil {
		t.Fatalf("registerHTTP error: %v", err)
	}
	for _, m := range []*probeModule{&self.probeModule, prot, pub} {
		if !m.registered {
			t.Fatalf("module %q was not registered", m.name)
		}
	}

	cases := []struct {
		path      string
		protected bool
	}{
		{"/api/v1/probe-prot/probe", true},
		{"/api/v1/probe-self/probe", false},
		{"/api/v1/probe-pub/probe", false},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		router.ServeHTTP(rec, req)
		got := rec.Header().Get("X-Protected") == "1"
		if got != c.protected {
			t.Fatalf("path %s protected = %v, want %v", c.path, got, c.protected)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("path %s status = %d, want %d", c.path, rec.Code, http.StatusNoContent)
		}
	}
}

func TestRegisterHTTPFailsClosedWithoutProtectedProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}

	err := registerHTTP(router, nil, nil, prot)
	if err == nil {
		t.Fatal("expected error when a protected capability has no middleware provider")
	}
	if !strings.Contains(err.Error(), "MountProtected") {
		t.Fatalf("error = %v, want it to name the mount requirement", err)
	}
	if prot.registered {
		t.Fatalf("capability %q must not be registered when no middleware provider exists", prot.name)
	}
}

// extraProtected（租户限流/幂等）非空不能替代认证中间件：受保护能力仍须拒绝启动。
func TestRegisterHTTPFailsClosedWhenExtraProtectedIsPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}
	extra := []gin.HandlerFunc{func(c *gin.Context) { c.Next() }}

	err := registerHTTP(router, nil, extra, prot)
	if err == nil {
		t.Fatal("expected error when extra protected middleware masks a missing provider")
	}
	if !strings.Contains(err.Error(), "MountProtected") {
		t.Fatalf("error = %v, want it to name the mount requirement", err)
	}
	if prot.registered {
		t.Fatalf("capability %q must not be registered when no middleware provider exists", prot.name)
	}
}
