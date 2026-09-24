package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminRoutesParity 钉住 P1.7 拆分后 /api/v1/admin/* 的对外路由集合：
// 与拆分前逐条一致，且无重复注册（gin 重复注册会 panic，此处再显式断言计数）。
func TestAdminRoutesParity(t *testing.T) {
	requireCapabilities(t, "user", "apikey", "queue", "dataops", "audit", "feature", "console")

	app := newTestAppWithDB(t)

	count := map[string]int{}
	for _, r := range app.router.Routes() {
		if len(r.Path) >= len("/api/v1/admin") && r.Path[:len("/api/v1/admin")] == "/api/v1/admin" {
			count[r.Method+" "+r.Path]++
		}
	}

	want := []string{
		http.MethodGet + " /api/v1/admin/error-codes",
		http.MethodGet + " /api/v1/admin/monitoring/status",
		http.MethodGet + " /api/v1/admin/monitoring/health",
		http.MethodGet + " /api/v1/admin/monitoring/metrics",
		http.MethodGet + " /api/v1/admin/ratelimit/auth",
		http.MethodGet + " /api/v1/admin/users",
		http.MethodPost + " /api/v1/admin/users",
		http.MethodGet + " /api/v1/admin/users/:id",
		http.MethodPut + " /api/v1/admin/users/:id",
		http.MethodDelete + " /api/v1/admin/users/:id",
		http.MethodPost + " /api/v1/admin/users/:id/roles",
		http.MethodGet + " /api/v1/admin/apikeys",
		http.MethodPost + " /api/v1/admin/apikeys",
		http.MethodGet + " /api/v1/admin/apikeys/:id",
		http.MethodDelete + " /api/v1/admin/apikeys/:id",
		http.MethodGet + " /api/v1/admin/config",
		http.MethodPut + " /api/v1/admin/config/:key",
		http.MethodPost + " /api/v1/admin/config/reload",
		http.MethodGet + " /api/v1/admin/tasks",
		http.MethodPost + " /api/v1/admin/tasks/:id/run",
		http.MethodPost + " /api/v1/admin/tasks/:id/toggle",
		http.MethodGet + " /api/v1/admin/tasks/:id/history",
		http.MethodPost + " /api/v1/admin/users/import/preview",
		http.MethodPost + " /api/v1/admin/users/import",
		http.MethodGet + " /api/v1/admin/users/import/template",
		http.MethodGet + " /api/v1/admin/users/import/:id",
		http.MethodGet + " /api/v1/admin/audit",
		http.MethodGet + " /api/v1/admin/jobs",
		http.MethodPost + " /api/v1/admin/jobs",
		http.MethodGet + " /api/v1/admin/jobs/:id",
		http.MethodPost + " /api/v1/admin/jobs/:id/retry",
		http.MethodGet + " /api/v1/admin/jobs/dead-letters",
		http.MethodPost + " /api/v1/admin/jobs/dead-letters/:id/resolve",
		http.MethodGet + " /api/v1/admin/ws",
		http.MethodPost + " /api/v1/admin/ws/push",
		http.MethodGet + " /api/v1/admin/ws/presence/:userId",
		http.MethodGet + " /api/v1/admin/ws/online",
		http.MethodGet + " /api/v1/admin/features",
		http.MethodPut + " /api/v1/admin/features/:name",
	}
	for _, route := range want {
		require.Equal(t, 1, count[route], "路由应恰好注册一次：%s（实际 %d 次）", route, count[route])
	}
}
