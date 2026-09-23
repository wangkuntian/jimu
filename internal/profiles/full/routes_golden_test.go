package full

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fullRoutesGolden 是 full 形态的完整对外路由集合（"METHOD path"，已排序）。
//
// 生成方式：把 full.Assembly() 在零值内核件上试运行（assembly.ProbeAssembly）得到的
// Module 注册进裸 *gin.Engine —— 与 tools/composereport 的 routeCount 同一机制 —— 再取
// r.Routes() 的 method+path 排序。运行时组合根对 MountProtected 能力用空 relativePath 的
// router.Group 挂载，不改变路径，因此本集合与实际启动逐条一致。
//
// 这是 full 零退化护栏的最后一环：P2.6 起 e2e 已按形态装配（internal/e2e 经 assembly.WireFor
// 与 active.Assembly()），本 golden 仍以逐值方式钉住 full 的对外路由面 ——
// 任何一条路由从 full 形态消失（或 Mount 误分类导致挂载点变化）都必须让本用例失败。
// 其它形态的路由 golden 见 internal/profiles/registry/routes_golden_test.go。
var fullRoutesGolden = []string{
	"DELETE /api/v1/admin/apikeys/:id",
	"DELETE /api/v1/admin/files",
	"DELETE /api/v1/admin/users/:id",
	"DELETE /api/v1/auth/devices",
	"DELETE /api/v1/auth/devices/:id",
	"DELETE /api/v1/auth/webauthn/credentials/:id",
	"DELETE /api/v1/permissions/:id",
	"DELETE /api/v1/roles/:id",
	"DELETE /api/v1/tenant-plans/:id",
	"DELETE /api/v1/tenants/:id",
	"DELETE /api/v1/users/:id",
	"GET /api/v1/admin/apikeys",
	"GET /api/v1/admin/apikeys/:id",
	"GET /api/v1/admin/audit",
	"GET /api/v1/admin/config",
	"GET /api/v1/admin/error-codes",
	"GET /api/v1/admin/features",
	"GET /api/v1/admin/jobs",
	"GET /api/v1/admin/jobs/:id",
	"GET /api/v1/admin/jobs/dead-letters",
	"GET /api/v1/admin/monitoring/health",
	"GET /api/v1/admin/monitoring/metrics",
	"GET /api/v1/admin/monitoring/status",
	"GET /api/v1/admin/ratelimit/auth",
	"GET /api/v1/admin/tasks",
	"GET /api/v1/admin/tasks/:id/history",
	"GET /api/v1/admin/users",
	"GET /api/v1/admin/users/:id",
	"GET /api/v1/admin/users/import/:id",
	"GET /api/v1/admin/users/import/template",
	"GET /api/v1/admin/ws",
	"GET /api/v1/admin/ws/online",
	"GET /api/v1/admin/ws/presence/:userId",
	"GET /api/v1/audits",
	"GET /api/v1/audits/:id",
	"GET /api/v1/audits/export",
	"GET /api/v1/audits/verify",
	"GET /api/v1/auth/devices",
	"GET /api/v1/auth/login-history",
	"GET /api/v1/auth/webauthn/credentials",
	"GET /api/v1/captcha",
	"GET /api/v1/oauth/:provider/callback",
	"GET /api/v1/oauth/:provider/login",
	"GET /api/v1/permissions",
	"GET /api/v1/permissions/:id",
	"GET /api/v1/roles",
	"GET /api/v1/roles/:id",
	"GET /api/v1/tenant-plans",
	"GET /api/v1/tenants",
	"GET /api/v1/tenants/:id",
	"GET /api/v1/tenants/usage",
	"GET /api/v1/users",
	"GET /api/v1/users/:id",
	"GET /api/v1/users/export.csv",
	"GET /swagger/*any",
	"POST /api/v1/admin/apikeys",
	"POST /api/v1/admin/config/reload",
	"POST /api/v1/admin/files",
	"POST /api/v1/admin/jobs",
	"POST /api/v1/admin/jobs/:id/retry",
	"POST /api/v1/admin/jobs/dead-letters/:id/resolve",
	"POST /api/v1/admin/tasks/:id/run",
	"POST /api/v1/admin/tasks/:id/toggle",
	"POST /api/v1/admin/users",
	"POST /api/v1/admin/users/:id/roles",
	"POST /api/v1/admin/users/import",
	"POST /api/v1/admin/users/import/preview",
	"POST /api/v1/admin/ws/push",
	"POST /api/v1/auth/forgot-password",
	"POST /api/v1/auth/login",
	"POST /api/v1/auth/logout",
	"POST /api/v1/auth/logout-all",
	"POST /api/v1/auth/mfa/disable",
	"POST /api/v1/auth/mfa/enable",
	"POST /api/v1/auth/mfa/setup",
	"POST /api/v1/auth/refresh",
	"POST /api/v1/auth/register",
	"POST /api/v1/auth/reset-password",
	"POST /api/v1/auth/webauthn/login/begin",
	"POST /api/v1/auth/webauthn/login/finish",
	"POST /api/v1/auth/webauthn/register/begin",
	"POST /api/v1/auth/webauthn/register/finish",
	"POST /api/v1/permissions",
	"POST /api/v1/roles",
	"POST /api/v1/roles/:id/permissions",
	"POST /api/v1/tenant-plans",
	"POST /api/v1/tenants",
	"POST /api/v1/users",
	"POST /api/v1/users/batch-delete",
	"PUT /api/v1/admin/config/:key",
	"PUT /api/v1/admin/features/:name",
	"PUT /api/v1/admin/users/:id",
	"PUT /api/v1/auth/webauthn/credentials/:id",
	"PUT /api/v1/permissions/:id",
	"PUT /api/v1/roles/:id",
	"PUT /api/v1/tenant-plans/:id",
	"PUT /api/v1/tenants/:id",
	"PUT /api/v1/tenants/:id/plan",
	"PUT /api/v1/users/:id",
}

// fullMountsGolden 是 full 形态 25 个条目的归一化挂载点。零值/未识别取值会被
// Descriptor.Normalized() 折叠为 MountProtected（fail-closed），因此误分类只可能表现为
// 期望 public/self-managed 的条目落回 protected —— 本表逐条钉住。
var fullMountsGolden = map[string]contract.MountPoint{
	"encryption":   contract.MountProtected,
	"storage":      contract.MountProtected,
	"notification": contract.MountProtected,
	"queue":        contract.MountProtected,
	"outbox":       contract.MountProtected,
	"breach":       contract.MountProtected,
	"tenant":       contract.MountProtected,
	"access":       contract.MountProtected,
	"user":         contract.MountProtected,
	"captcha":      contract.MountPublic,
	"mfa":          contract.MountSelfManaged,
	"auth":         contract.MountSelfManaged,
	"passkey":      contract.MountSelfManaged,
	"audit":        contract.MountProtected,
	"console":      contract.MountSelfManaged,
	"oauth":        contract.MountPublic,
	"apikey":       contract.MountProtected,
	"dataops":      contract.MountProtected,
	"feature":      contract.MountProtected,
	"uploadsec":    contract.MountProtected,
	"search":       contract.MountProtected,
	"retention":    contract.MountProtected,
	"apidocs":      contract.MountPublic,
	"grpc":         contract.MountProtected,
	"ws":           contract.MountProtected,
}

// TestFullAssemblyRoutesGolden 钉住 full 形态的路由集合：多一条、少一条、路径或方法变化
// 都会失败。集合来自真实装配路径（Assembly → ProbeAssembly → RegisterHTTP），不是手写清单。
func TestFullAssemblyRoutesGolden(t *testing.T) {
	got := probeRoutes(t)
	require.Equal(t, len(fullRoutesGolden), len(got),
		"full 形态路由数变化：期望 %d，实际 %d\n实际集合：\n%s",
		len(fullRoutesGolden), len(got), strings.Join(got, "\n"))
	require.Equal(t, fullRoutesGolden, got)
	require.Contains(t, got, http.MethodGet+" /api/v1/users")
}

// TestFullAssemblyMountsGolden 钉住 full 形态每个条目的归一化挂载点，
// 防止 MountProtected 误分类静默取消路由保护。
func TestFullAssemblyMountsGolden(t *testing.T) {
	caps := Assembly().Capabilities
	require.Len(t, caps, len(fullMountsGolden), "full 形态条目数变化，挂载点表需同步更新")
	for _, c := range caps {
		want, ok := fullMountsGolden[c.Descriptor.Name]
		require.True(t, ok, "未登记挂载点的条目 %q", c.Descriptor.Name)
		require.Equal(t, want, c.Descriptor.Normalized(), "能力 %q 的挂载点变化", c.Descriptor.Name)
	}
}

// probeRoutes 复现 tools/composereport 的路由探测：零值内核件上试运行 Wire，把非空 Module
// 注册进裸 gin.Engine，返回排序后的 "METHOD path"。
func probeRoutes(t *testing.T) []string {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)
	res, err := assembly.ProbeAssembly(Assembly(), nil)
	require.NoError(t, err)

	r := gin.New()
	for _, m := range res.Modules {
		m.RegisterHTTP(r)
	}
	out := make([]string, 0, len(r.Routes()))
	for _, rt := range r.Routes() {
		out = append(out, rt.Method+" "+rt.Path)
	}
	sort.Strings(out)
	return out
}
