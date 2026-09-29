package catalog

import (
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/config"
	"jimu/internal/contract"
)

// withEntries 用测试夹具临时替换能力清单，测试结束自动恢复。
func withEntries(t *testing.T, ds ...contract.Descriptor) {
	t.Helper()
	old := entries
	entries = ds
	t.Cleanup(func() { entries = old })
}

// fixture 复刻真实清单（8 业务能力 + 5 基础设施能力）的依赖形态。
// Migrations 为 fs.FS 接口值（embed.FS 无法逐值复刻），漂移检测由
// TestCatalogMigrationsShape 单独按"有无迁移"钉住。
func fixture() []contract.Descriptor {
	return []contract.Descriptor{
		{Name: "user", SoftRequires: []string{"access", "tenant"},
			Owns: []string{"users"}, Mount: contract.MountProtected},
		{Name: "access", Requires: []string{"user"}, SoftRequires: []string{"tenant"},
			Owns:  []string{"roles", "permissions", "role_permissions", "user_roles"},
			Mount: contract.MountProtected},
		{Name: "tenant", Requires: []string{"user", "access"},
			Owns: []string{"tenants", "tenant_plans"}, Mount: contract.MountProtected},
		{Name: "mfa", Requires: []string{"user"}, SoftRequires: []string{"auth"},
			Owns: []string{"user_mfa", "trusted_devices"}, Mount: contract.MountSelfManaged},
		{Name: "auth", Requires: []string{"user", "access"},
			SoftRequires: []string{"tenant", "mfa", "captcha", "breach"},
			Owns:         []string{"login_histories", "password_histories"},
			Mount:        contract.MountSelfManaged},
		{Name: "passkey", Requires: []string{"user", "auth"},
			Owns: []string{"webauthn_credentials"}, Mount: contract.MountSelfManaged},
		{Name: "audit", Owns: []string{"audit_logs", "audit_chain_head"}, Mount: contract.MountProtected},
		{Name: "console", Requires: []string{"auth", "access"}, Mount: contract.MountSelfManaged},
		{Name: "oauth", Requires: []string{"auth", "user"},
			Owns: []string{"user_oauth_bindings"}, Mount: contract.MountPublic},
		{Name: "apikey", SoftRequires: []string{"tenant"},
			Owns: []string{"api_keys"}, Mount: contract.MountProtected},
		{Name: "queue", Owns: []string{"jobs", "job_history", "dead_letters", "scheduled_jobs"},
			Drivers: []string{"redis", "kafka", "rabbitmq"}, Mount: contract.MountProtected},
		{Name: "outbox", SoftRequires: []string{"queue"},
			Owns: []string{"outbox_events"}, Mount: contract.MountProtected},
		{Name: "dataops", Owns: []string{"import_jobs"},
			Drivers: []string{"csv", "excel"}, Mount: contract.MountProtected},
		{Name: "search", Owns: []string{"search_documents"}, Mount: contract.MountProtected},
		{Name: "captcha", Mount: contract.MountPublic},
		{Name: "feature", Mount: contract.MountProtected},
		{Name: "uploadsec", Mount: contract.MountProtected},
		{Name: "breach", Mount: contract.MountProtected},
	}
}

func TestAllReturnsDeepCopyOfRequires(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "a", Requires: []string{"b"}}, contract.Descriptor{Name: "b"})
	got := All()
	got[0].Requires[0] = "mutated"
	if All()[0].Requires[0] != "b" {
		t.Fatal("All() must not expose the registry's Requires backing array")
	}
}

// TestAllReturnsDeepCopyOfSoftRequiresAndOwns 软依赖与自有表同样不得暴露清单底层数组。
func TestAllReturnsDeepCopyOfSoftRequiresAndOwns(t *testing.T) {
	withEntries(t, contract.Descriptor{
		Name:         "a",
		SoftRequires: []string{"b"},
		Owns:         []string{"t1"},
	}, contract.Descriptor{Name: "b"})
	got := All()
	got[0].SoftRequires[0] = "mutated"
	got[0].Owns[0] = "mutated"
	assert.Equal(t, "b", All()[0].SoftRequires[0], "All() must not expose the registry's SoftRequires backing array")
	assert.Equal(t, "t1", All()[0].Owns[0], "All() must not expose the registry's Owns backing array")
}

func TestAllReturnsDeepCopyOfPermissions(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "a", Permissions: []contract.Permission{{Name: "p", Resource: "/r", Action: "GET"}}})
	got := All()
	got[0].Permissions[0].Resource = "mutated"
	if All()[0].Permissions[0].Resource != "/r" {
		t.Fatal("All() must not expose the registry's Permissions backing array")
	}
}

// TestAllReturnsDeepCopyOfDrivers 驱动声明同样不得暴露清单底层数组
// （queue/dataops 是当前仅有的两个声明驱动的能力）。
func TestAllReturnsDeepCopyOfDrivers(t *testing.T) {
	withEntries(t,
		contract.Descriptor{Name: "queue", Drivers: []string{"redis", "kafka", "rabbitmq"}},
		contract.Descriptor{Name: "dataops", Drivers: []string{"csv", "excel"}},
	)
	got := All()
	got[0].Drivers[0] = "mutated"
	got[1].Drivers[0] = "mutated"
	again := All()
	assert.Equal(t, "redis", again[0].Drivers[0], "All() must not expose the registry's Drivers backing array")
	assert.Equal(t, "csv", again[1].Drivers[0], "All() must not expose the registry's Drivers backing array")
}

// TestValidateDeclarationsAcceptsCurrentCatalog 真实清单必须通过声明校验。
func TestValidateDeclarationsAcceptsCurrentCatalog(t *testing.T) {
	require.NoError(t, ValidateDeclarations())
}

// TestDescriptorsAreWellFormed 同时校验夹具与真实清单。
func TestDescriptorsAreWellFormed(t *testing.T) {
	lists := map[string][]contract.Descriptor{"fixture": fixture(), "catalog": All()}
	for label, list := range lists {
		known := make(map[string]bool, len(list))
		for _, d := range list {
			known[d.Name] = true
		}
		for _, d := range list {
			if d.Name == "" {
				t.Fatalf("%s: capability has empty name", label)
			}
			for _, dep := range d.Requires {
				if !known[dep] {
					t.Fatalf("%s: capability %q requires unknown capability %q", label, d.Name, dep)
				}
			}
			switch d.Normalized() {
			case contract.MountPublic, contract.MountProtected, contract.MountSelfManaged:
			default:
				t.Fatalf("%s: capability %q has invalid mount %q", label, d.Name, d.Mount)
			}
		}
		index := make(map[string]int, len(list))
		for i, d := range list {
			if _, dup := index[d.Name]; dup {
				t.Fatalf("%s: duplicate capability name %q", label, d.Name)
			}
			index[d.Name] = i
		}
		for _, d := range list {
			for _, dep := range d.Requires {
				if index[dep] >= index[d.Name] {
					t.Fatalf("%s: capability %q at index %d must follow its dependency %q at index %d",
						label, d.Name, index[d.Name], dep, index[dep])
				}
			}
		}
	}
	// 精确逐值比较对 fs.FS（embed.FS）与 ConfigSpec.New（函数值）不可行，比较除
	// Migrations/Configs 外的全部字段；迁移有无形态由 TestCatalogMigrationsShape 单独钉住，
	// 配置段形态由 TestCatalogConfigSectionsShape 单独钉住；权限点聚合面由
	// TestDescriptorPermissionsCoverBusinessRoutes 单独钉住。
	got, want := All(), fixture()
	for i := range want {
		got[i].Migrations, want[i].Migrations = nil, nil
		got[i].Permissions, want[i].Permissions = nil, nil
		got[i].Configs, want[i].Configs = nil, nil
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog descriptors drifted from the expected fixture:\n got %+v\nwant %+v", got, want)
	}
}

// migrationsOf 便于漂移比较：embed.FS 无法逐值构造，只比较"有无迁移"。
func migrationsOf(ds []contract.Descriptor) map[string]bool {
	out := make(map[string]bool, len(ds))
	for _, d := range ds {
		out[d.Name] = d.Migrations != nil
	}
	return out
}

// TestCatalogMigrationsShape 钉住：除 admin 外的清单能力都必须自带迁移，
// admin（无迁移）必须为 nil。
func TestCatalogMigrationsShape(t *testing.T) {
	want := map[string]bool{
		"user": true, "access": true, "tenant": true,
		"mfa": true, "auth": true, "passkey": true, "audit": true, "oauth": true,
		"console": false,
		"captcha": false, "breach": false, "feature": false, "uploadsec": false,
		"apikey": true, "queue": true, "outbox": true, "dataops": true, "search": true,
	}
	if got := migrationsOf(All()); !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog Migrations shape drifted:\n got %v\nwant %v", got, want)
	}
}

// TestCatalogOwnsShape 钉住各能力拥有的表（空 = 不拥有表）。
func TestCatalogOwnsShape(t *testing.T) {
	want := map[string][]string{
		"user":    {"users"},
		"access":  {"roles", "permissions", "role_permissions", "user_roles"},
		"tenant":  {"tenants", "tenant_plans"},
		"mfa":     {"user_mfa", "trusted_devices"},
		"auth":    {"login_histories", "password_histories"},
		"passkey": {"webauthn_credentials"},
		"oauth":   {"user_oauth_bindings"},
		"apikey":  {"api_keys"},
		"queue":   {"jobs", "job_history", "dead_letters", "scheduled_jobs"},
		"outbox":  {"outbox_events"},
		"dataops": {"import_jobs"},
		"search":  {"search_documents"},
		"audit":   {"audit_logs", "audit_chain_head"},
	}
	got := map[string][]string{}
	for _, d := range All() {
		if len(d.Owns) > 0 {
			got[d.Name] = d.Owns
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog Owns drifted:\n got %v\nwant %v", got, want)
	}
}

// TestCatalogSoftRequiresShape 钉住可选依赖（只登记代码里真实可降级的耦合）。
func TestCatalogSoftRequiresShape(t *testing.T) {
	want := map[string][]string{
		"user":   {"access", "tenant"},
		"access": {"tenant"},
		"mfa":    {"auth"},
		"auth":   {"tenant", "mfa", "captcha", "breach"},
		"apikey": {"tenant"},
		"outbox": {"queue"},
	}
	got := map[string][]string{}
	for _, d := range All() {
		if len(d.SoftRequires) > 0 {
			got[d.Name] = d.SoftRequires
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog SoftRequires drifted:\n got %v\nwant %v", got, want)
	}
}

// TestCatalogDriversShape 钉住各能力声明的驱动可用集（空 = 无驱动概念）。
// 驱动目录是否存在、形态选中集是否与 import 闭包一致，由 make check-capabilities 校验。
func TestCatalogDriversShape(t *testing.T) {
	want := map[string][]string{
		"queue":   {"redis", "kafka", "rabbitmq"},
		"dataops": {"csv", "excel"},
	}
	got := map[string][]string{}
	for _, d := range All() {
		if len(d.Drivers) > 0 {
			got[d.Name] = d.Drivers
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog Drivers drifted:\n got %v\nwant %v", got, want)
	}
}

// TestCatalogAssetsShape 钉住各能力声明的非代码资产（P2.6）。
// 当前只有非 catalog 的 apidocs 声明资产（docs/openapi），因此 catalog 能力必须全为空 ——
// 若将来给 catalog 能力加资产，这里会红，提醒同步 TestDescriptorsAreWellFormed 的逐值 fixture。
// 资产归属唯一性与「无未声明资产」由 make check-capabilities 的资产段校验。
func TestCatalogAssetsShape(t *testing.T) {
	for _, d := range All() {
		if len(d.Assets) > 0 {
			t.Fatalf("catalog capability %q declares assets %v: 请同步 catalog_test.go 的 fixture() 与 TestCatalogAssetsShape", d.Name, d.Assets)
		}
	}
}

// TestTenantRequiresUserAndRole：tenant 的 005_tenants.sql 会 ALTER users/roles，
// 迁移执行顺序由 Resolve 闭包序保证，故 tenant.Requires 必须含 user 与 access（roles 表所有者）。
func TestTenantRequiresUserAndRole(t *testing.T) {
	for _, d := range All() {
		if d.Name != "tenant" {
			continue
		}
		if !slices.Contains(d.Requires, "user") || !slices.Contains(d.Requires, "access") {
			t.Fatalf("tenant.Requires = %v, want to contain \"user\" and \"access\"", d.Requires)
		}
	}
}

// TestDescriptorPermissionsCoverBusinessRoutes 钉住能力声明的权限点聚合面：
// 迁移后权限点改由 Descriptor 声明，聚合结果必须逐值等于全部 45 个权限点
// （user 5 + access 11 + audit 3 + tenant 9 + console 4 + mfa 6 + passkey 7），
// 既不缺失也不多出。
func TestDescriptorPermissionsCoverBusinessRoutes(t *testing.T) {
	required := []struct{ resource, action string }{
		{"/api/v1/users", "GET"}, {"/api/v1/users", "POST"},
		{"/api/v1/users/*", "GET"}, {"/api/v1/users/*", "PUT"}, {"/api/v1/users/*", "DELETE"},
		{"/api/v1/roles", "GET"}, {"/api/v1/roles", "POST"},
		{"/api/v1/roles/*", "GET"}, {"/api/v1/roles/*", "PUT"}, {"/api/v1/roles/*", "DELETE"},
		{"/api/v1/roles/*/permissions", "POST"},
		{"/api/v1/permissions", "GET"}, {"/api/v1/permissions", "POST"},
		{"/api/v1/permissions/*", "GET"}, {"/api/v1/permissions/*", "PUT"}, {"/api/v1/permissions/*", "DELETE"},
		{"/api/v1/audits", "GET"}, {"/api/v1/audits/*", "GET"}, {"/api/v1/audits/export", "GET"},
		{"/api/v1/tenants", "GET"}, {"/api/v1/tenants", "POST"},
		{"/api/v1/tenants/*", "GET"}, {"/api/v1/tenants/*", "PUT"}, {"/api/v1/tenants/*", "DELETE"},
		{"/api/v1/tenant-plans", "GET"}, {"/api/v1/tenant-plans", "POST"},
		{"/api/v1/tenant-plans/*", "PUT"}, {"/api/v1/tenant-plans/*", "DELETE"},
		{"/api/v1/admin/*", "GET"}, {"/api/v1/admin/*", "POST"},
		{"/api/v1/admin/*", "PUT"}, {"/api/v1/admin/*", "DELETE"},
		{"/api/v1/auth/mfa/setup", "POST"}, {"/api/v1/auth/mfa/enable", "POST"},
		{"/api/v1/auth/mfa/disable", "POST"},
		{"/api/v1/auth/devices", "GET"}, {"/api/v1/auth/devices", "DELETE"},
		{"/api/v1/auth/devices/*", "DELETE"},
		{"/api/v1/auth/webauthn/login/begin", "POST"}, {"/api/v1/auth/webauthn/login/finish", "POST"},
		{"/api/v1/auth/webauthn/register/begin", "POST"}, {"/api/v1/auth/webauthn/register/finish", "POST"},
		{"/api/v1/auth/webauthn/credentials", "GET"},
		{"/api/v1/auth/webauthn/credentials/*", "PUT"}, {"/api/v1/auth/webauthn/credentials/*", "DELETE"},
	}
	got := map[string]bool{}
	total := 0
	for _, d := range All() {
		for _, p := range d.Permissions {
			got[p.Resource+" "+p.Action] = true
			total++
		}
	}
	if total != len(required) {
		t.Fatalf("aggregated permission points = %d, want %d", total, len(required))
	}
	for _, item := range required {
		if !got[item.resource+" "+item.action] {
			t.Fatalf("missing permission %s %s", item.action, item.resource)
		}
	}
}

// TestCatalogConfigSectionsShape 钉住各能力声明的配置段（ConfigSpec.New 是函数值，
// 无法参与 TestDescriptorsAreWellFormed 的逐值比较，故单独钉住段键与 SectionConfig 契约）。
// 非 catalog 包（storage/notification/retention）的段由组合根显式加载，此处不出现（③ 裁定 B）。
func TestCatalogConfigSectionsShape(t *testing.T) {
	want := map[string][]string{
		"auth":      {"auth"},
		"captcha":   {"captcha"},
		"audit":     {"audit"},
		"oauth":     {"oauth"},
		"queue":     {"queue", "scheduler"},
		"outbox":    {"outbox"},
		"uploadsec": {"upload"},
	}
	got := make(map[string][]string, len(want))
	for _, d := range All() {
		for _, c := range d.Configs {
			got[d.Name] = append(got[d.Name], c.Section)
			if c.New == nil {
				t.Fatalf("capability %q config %q has no constructor", d.Name, c.Section)
			}
			if _, ok := c.New().(config.SectionConfig); !ok {
				t.Fatalf("capability %q config %q must implement config.SectionConfig", d.Name, c.Section)
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog config sections drifted:\n got %v\nwant %v", got, want)
	}
}

// TestResolveAuthDoesNotPullTenantOrMFA auth 的 tenant/mfa 是软依赖：显式启用 auth
// 只补齐硬依赖 user/access，不再自动带入租户与二次验证（minimal profile 的前提）。
func TestResolveAuthDoesNotPullTenantOrMFA(t *testing.T) {
	got, err := Resolve([]string{"auth"})
	require.NoError(t, err)

	names := make([]string, 0, len(got))
	for _, d := range got {
		names = append(names, d.Name)
	}
	assert.Equal(t, []string{"user", "access", "auth"}, names)
	assert.NotContains(t, names, "tenant")
	assert.NotContains(t, names, "mfa")
}

// TestDegradedWithRealCatalog 关闭软依赖后对应能力出现在降级清单里。
// 解析与降级算法在 internal/capability，此处钉住 catalog 薄封装对真实清单的行为。
func TestDegradedWithRealCatalog(t *testing.T) {
	// 只启用 auth 的硬依赖闭包（user/access），tenant/mfa/captcha/breach 缺席
	caps, err := Resolve([]string{"auth"})
	require.NoError(t, err)
	got := Degraded(caps)
	require.Len(t, got, 3, "user/access/auth 声明了缺失的软依赖")
	assert.Equal(t, "user", got[0].Capability)
	assert.Equal(t, []string{"tenant"}, got[0].Missing)
	assert.Equal(t, "access", got[1].Capability)
	assert.Equal(t, []string{"tenant"}, got[1].Missing)
	assert.Equal(t, "auth", got[2].Capability)
	assert.Equal(t, []string{"tenant", "mfa", "captcha", "breach"}, got[2].Missing)
}

// TestDegradedNoneOnFullCatalog 全部启用（默认）时无降级项。
func TestDegradedNoneOnFullCatalog(t *testing.T) {
	assert.Empty(t, Degraded(All()))
}
