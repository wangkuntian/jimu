package catalog

import (
	"reflect"
	"slices"
	"strings"
	"testing"

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
		{Name: "user", Mount: contract.MountProtected},
		{Name: "access", Requires: []string{"user"}, Mount: contract.MountProtected},
		{Name: "tenant", Requires: []string{"user", "access"}, Mount: contract.MountProtected},
		{Name: "mfa", Requires: []string{"user"}, Mount: contract.MountSelfManaged},
		{Name: "auth", Requires: []string{"user", "access", "tenant", "mfa"}, Mount: contract.MountSelfManaged},
		{Name: "passkey", Requires: []string{"user", "auth"}, Mount: contract.MountSelfManaged},
		{Name: "audit", Mount: contract.MountProtected},
		{Name: "console", Requires: []string{"auth", "access"}, Mount: contract.MountSelfManaged},
		{Name: "oauth", Requires: []string{"auth", "user"}, Mount: contract.MountPublic},
		{Name: "apikey", Mount: contract.MountProtected},
		{Name: "queue", Mount: contract.MountProtected},
		{Name: "outbox", Mount: contract.MountProtected},
		{Name: "dataops", Mount: contract.MountProtected},
		{Name: "search", Mount: contract.MountProtected},
		{Name: "captcha", Mount: contract.MountPublic},
		{Name: "feature", Mount: contract.MountProtected},
		{Name: "uploadsec", Mount: contract.MountProtected},
		{Name: "breach", Mount: contract.MountProtected},
	}
}

func TestResolveEmptyMeansAll(t *testing.T) {
	withEntries(t, fixture()...)
	got, err := Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve(nil) error: %v", err)
	}
	if len(got) != 18 {
		t.Fatalf("len = %d, want 18 (all)", len(got))
	}
}

func TestResolveUnknownCapability(t *testing.T) {
	withEntries(t, fixture()...)
	_, err := Resolve([]string{"nope"})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
	if !strings.Contains(err.Error(), "unknown capability") {
		t.Fatalf("error = %v, want unknown capability", err)
	}
}

func TestResolvePullsDependenciesAndKeepsOrder(t *testing.T) {
	withEntries(t, fixture()...)
	// access 依赖 user，user 无依赖；顺序必须与清单一致（依赖在前）
	got, err := Resolve([]string{"access"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "user" || got[1].Name != "access" {
		t.Fatalf("Resolve([access]) = %v, want [user access]", namesOf(got))
	}
}

func TestResolveClosureIsTransitive(t *testing.T) {
	withEntries(t, fixture()...)
	// oauth -> auth -> access/tenant -> user
	got, err := Resolve([]string{"oauth"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	want := map[string]bool{"oauth": true, "auth": true, "mfa": true, "user": true, "access": true, "tenant": true}
	if len(got) != len(want) {
		t.Fatalf("Resolve([oauth]) = %v, want %d entries", namesOf(got), len(want))
	}
	for _, d := range got {
		if !want[d.Name] {
			t.Fatalf("unexpected capability %q in closure %v", d.Name, namesOf(got))
		}
	}
	wantOrder := []string{"user", "access", "tenant", "mfa", "auth", "oauth"}
	if gotOrder := namesOf(got); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("Resolve([oauth]) order = %v, want %v", gotOrder, wantOrder)
	}
}

func TestResolveIsSubsetOfAll(t *testing.T) {
	withEntries(t, fixture()...)
	got, err := Resolve([]string{"user"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "user" {
		t.Fatalf("Resolve([user]) = %v, want [user]", namesOf(got))
	}
}

func TestResolveRejectsUnknownDependency(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "broken", Requires: []string{"ghost"}})
	_, err := Resolve([]string{"broken"})
	if err == nil || !strings.Contains(err.Error(), `requires unknown capability "ghost"`) {
		t.Fatalf("error = %v, want requires unknown capability", err)
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

func TestResolveReturnsDeepCopyOfRequires(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "b"}, contract.Descriptor{Name: "a", Requires: []string{"b"}})
	got, err := Resolve([]string{"a"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	got[1].Requires[0] = "mutated"
	if again, err := Resolve([]string{"a"}); err != nil || again[1].Requires[0] != "b" {
		t.Fatalf("Resolve() must not expose the registry's Requires backing array (err = %v)", err)
	}
}

func TestAllReturnsDeepCopyOfPermissions(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "a", Permissions: []contract.Permission{{Name: "p", Resource: "/r", Action: "GET"}}})
	got := All()
	got[0].Permissions[0].Resource = "mutated"
	if All()[0].Permissions[0].Resource != "/r" {
		t.Fatal("All() must not expose the registry's Permissions backing array")
	}
}

func TestResolveReturnsDeepCopyOfPermissions(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "a", Permissions: []contract.Permission{{Name: "p", Resource: "/r", Action: "GET"}}})
	got, err := Resolve([]string{"a"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	got[0].Permissions[0].Resource = "mutated"
	if again, err := Resolve([]string{"a"}); err != nil || again[0].Permissions[0].Resource != "/r" {
		t.Fatalf("Resolve() must not expose the registry's Permissions backing array (err = %v)", err)
	}
}

func TestResolveReportsFirstDanglingDependencyInListOrder(t *testing.T) {
	// 两个坏依赖：错误必须稳定指向清单顺序里的第一个，而不是 map 遍历的随机一个
	withEntries(t,
		contract.Descriptor{Name: "a", Requires: []string{"ghost-a"}},
		contract.Descriptor{Name: "b", Requires: []string{"ghost-b"}},
	)
	for i := 0; i < 200; i++ {
		_, err := Resolve([]string{"a", "b"})
		if err == nil || !strings.Contains(err.Error(), `"ghost-a"`) {
			t.Fatalf("run %d: error = %v, want the first dangling dependency in list order", i, err)
		}
	}
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
	// 精确逐值比较对 fs.FS（embed.FS）不可行，比较除 Migrations 外的全部字段；
	// 迁移有无形态由 TestCatalogMigrationsShape 单独钉住；权限点聚合面由
	// TestDescriptorPermissionsCoverBusinessRoutes 单独钉住。
	got, want := All(), fixture()
	for i := range want {
		got[i].Migrations, want[i].Migrations = nil, nil
		got[i].Permissions, want[i].Permissions = nil, nil
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

func namesOf(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
