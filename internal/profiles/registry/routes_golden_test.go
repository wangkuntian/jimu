package registry

import (
	"os"
	"sort"
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// shapeRoutesGolden 钉住各**非 full** 形态的完整对外路由集合（"METHOD path"，已排序）。
//
// 与 full 的 golden（internal/profiles/full/routes_golden_test.go）同一机制：把形态清单在
// 零值内核件上试运行（assembly.ProbeAssembly，与 tools/composereport 的 routeCount 同源），
// 把非空 Module 注册进裸 *gin.Engine，取 r.Routes() 的 method+path 排序。
//
// 为什么放在 registry 而不是每个形态各一份：形态清单的唯一来源就在这里，一处即可横向对照，
// 新增形态时只改本文件（full 保留自己的 golden，额外还钉挂载点）。
//
// 更新方式（改了路由后）：SHAPE_ROUTES_UPDATE=1 go test ./internal/profiles/registry/ -run TestShapeRoutesGolden -v
// 注意：更新模式会**故意 Fatalf**（打印后失败），以免该环境变量被误带进 CI 时静默跳过 golden 校验。
var shapeRoutesGolden = map[string][]string{
	"minimal": {
		"DELETE /api/v1/admin/users/:id",
		"DELETE /api/v1/permissions/:id",
		"DELETE /api/v1/roles/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/:id",
		"GET /api/v1/auth/login-history",
		"GET /api/v1/permissions",
		"GET /api/v1/permissions/:id",
		"GET /api/v1/roles",
		"GET /api/v1/roles/:id",
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"GET /api/v1/users/export.csv",
		"POST /api/v1/admin/users",
		"POST /api/v1/admin/users/:id/roles",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/logout-all",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/reset-password",
		"POST /api/v1/permissions",
		"POST /api/v1/roles",
		"POST /api/v1/roles/:id/permissions",
		"POST /api/v1/users",
		"POST /api/v1/users/batch-delete",
		"PUT /api/v1/admin/users/:id",
		"PUT /api/v1/permissions/:id",
		"PUT /api/v1/roles/:id",
		"PUT /api/v1/users/:id",
	},
	"saas": {
		"DELETE /api/v1/admin/users/:id",
		"DELETE /api/v1/permissions/:id",
		"DELETE /api/v1/roles/:id",
		"DELETE /api/v1/tenant-plans/:id",
		"DELETE /api/v1/tenants/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/admin/audit",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/:id",
		"GET /api/v1/audits",
		"GET /api/v1/audits/:id",
		"GET /api/v1/audits/export",
		"GET /api/v1/audits/verify",
		"GET /api/v1/auth/login-history",
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
		"POST /api/v1/admin/users",
		"POST /api/v1/admin/users/:id/roles",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/logout-all",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/reset-password",
		"POST /api/v1/permissions",
		"POST /api/v1/roles",
		"POST /api/v1/roles/:id/permissions",
		"POST /api/v1/tenant-plans",
		"POST /api/v1/tenants",
		"POST /api/v1/users",
		"POST /api/v1/users/batch-delete",
		"PUT /api/v1/admin/users/:id",
		"PUT /api/v1/permissions/:id",
		"PUT /api/v1/roles/:id",
		"PUT /api/v1/tenant-plans/:id",
		"PUT /api/v1/tenants/:id",
		"PUT /api/v1/tenants/:id/plan",
		"PUT /api/v1/users/:id",
	},
	"enterprise": {
		"DELETE /api/v1/admin/users/:id",
		"DELETE /api/v1/permissions/:id",
		"DELETE /api/v1/roles/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/admin/audit",
		"GET /api/v1/admin/config",
		"GET /api/v1/admin/error-codes",
		"GET /api/v1/admin/monitoring/health",
		"GET /api/v1/admin/monitoring/metrics",
		"GET /api/v1/admin/monitoring/status",
		"GET /api/v1/admin/ratelimit/auth",
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
		"GET /api/v1/auth/login-history",
		"GET /api/v1/oauth/:provider/callback",
		"GET /api/v1/oauth/:provider/login",
		"GET /api/v1/permissions",
		"GET /api/v1/permissions/:id",
		"GET /api/v1/roles",
		"GET /api/v1/roles/:id",
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"GET /api/v1/users/export.csv",
		"POST /api/v1/admin/config/reload",
		"POST /api/v1/admin/users",
		"POST /api/v1/admin/users/:id/roles",
		"POST /api/v1/admin/users/import",
		"POST /api/v1/admin/users/import/preview",
		"POST /api/v1/admin/ws/push",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/logout-all",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/reset-password",
		"POST /api/v1/permissions",
		"POST /api/v1/roles",
		"POST /api/v1/roles/:id/permissions",
		"POST /api/v1/users",
		"POST /api/v1/users/batch-delete",
		"PUT /api/v1/admin/config/:key",
		"PUT /api/v1/admin/users/:id",
		"PUT /api/v1/permissions/:id",
		"PUT /api/v1/roles/:id",
		"PUT /api/v1/users/:id",
	},
	"machine": {
		"DELETE /api/v1/admin/apikeys/:id",
		"DELETE /api/v1/admin/users/:id",
		"DELETE /api/v1/permissions/:id",
		"DELETE /api/v1/roles/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/admin/apikeys",
		"GET /api/v1/admin/apikeys/:id",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/:id",
		"GET /api/v1/permissions",
		"GET /api/v1/permissions/:id",
		"GET /api/v1/roles",
		"GET /api/v1/roles/:id",
		"GET /api/v1/users",
		"GET /api/v1/users/:id",
		"GET /api/v1/users/export.csv",
		"POST /api/v1/admin/apikeys",
		"POST /api/v1/admin/users",
		"POST /api/v1/admin/users/:id/roles",
		"POST /api/v1/permissions",
		"POST /api/v1/roles",
		"POST /api/v1/roles/:id/permissions",
		"POST /api/v1/users",
		"POST /api/v1/users/batch-delete",
		"PUT /api/v1/admin/users/:id",
		"PUT /api/v1/permissions/:id",
		"PUT /api/v1/roles/:id",
		"PUT /api/v1/users/:id",
	},
}

func TestShapeRoutesGolden(t *testing.T) {
	if os.Getenv("SHAPE_ROUTES_UPDATE") != "" {
		for _, name := range Names() {
			if name == "full" {
				continue // full 由自身包的 golden 钉住
			}
			t.Logf("\t%q: {", name)
			for _, r := range routesOf(t, name) {
				t.Logf("\t\t%q,", r)
			}
			t.Logf("\t},")
		}
		t.Fatalf("SHAPE_ROUTES_UPDATE=1：已打印各形态当前路由，请粘进 shapeRoutesGolden（此模式故意失败，避免它被误带进 CI 而静默跳过校验）")
	}

	for _, name := range Names() {
		if name == "full" {
			continue
		}
		want, ok := shapeRoutesGolden[name]
		require.Truef(t, ok, "形态 %s 未登记路由 golden", name)
		require.NotEmptyf(t, want, "形态 %s 的 golden 为空（请用 SHAPE_ROUTES_UPDATE=1 生成）", name)
		require.Equalf(t, want, routesOf(t, name), "形态 %s 的对外路由集合变化", name)
	}
}

// TestShapeMountsMatchFull 钉住跨形态一致性：同一能力在不同形态里的挂载点必须相同。
//
// 计划原本要求「四个非 full 形态各补一份挂载点 golden」；这里改为**一条跨形态断言**替代四份
// 近似重复的副本（各形态清单逐字复用能力包导出的 Descriptor，值本就相同）：它能抓到同一个
// 问题（某形态为某能力声明了不同 Mount，从而改变保护语义），且新增形态时无需再补一份数据。
// full 另有逐值 mounts golden（internal/profiles/full 的 TestFullAssemblyMountsGolden）。
func TestShapeMountsMatchFull(t *testing.T) {
	fullAsm, err := Lookup("full")
	require.NoError(t, err)

	want := make(map[string]contract.MountPoint, len(fullAsm.Capabilities))
	for _, c := range fullAsm.Capabilities {
		want[c.Descriptor.Name] = c.Descriptor.Normalized()
	}
	require.NotEmpty(t, want)

	for _, name := range Names() {
		if name == "full" {
			continue
		}
		asm, err := Lookup(name)
		require.NoError(t, err)
		for _, c := range asm.Capabilities {
			w, ok := want[c.Descriptor.Name]
			require.Truef(t, ok, "形态 %s 出现 full 未声明的能力 %q", name, c.Descriptor.Name)
			require.Equalf(t, w, c.Descriptor.Normalized(),
				"形态 %s 的能力 %q 挂载点与 full 不一致（可能改变路由保护语义）", name, c.Descriptor.Name)
		}
	}
}

// routesOf 复现 tools/composereport 的路由探测：零值内核件上试运行 Wire，把非空 Module
// 注册进裸 gin.Engine，返回排序后的 "METHOD path"（与 full 包的同名辅助逐字同构）。
func routesOf(t *testing.T, name string) []string {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)

	asm, err := Lookup(name)
	require.NoError(t, err)
	res, err := assembly.ProbeAssembly(asm, nil)
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
