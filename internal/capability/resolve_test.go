package capability

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
)

// fixture 复刻真实清单（8 业务能力 + 5 基础设施能力）的依赖形态，与
// catalog_test.go 的同名夹具逐值一致；解析算法只读 Name/Requires/SoftRequires，
// Owns 与 Mount 随之保留以保持形态可比。
// Migrations 为 fs.FS 接口值（embed.FS 无法逐值复刻），漂移检测由 catalog 的
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
			Mount: contract.MountProtected},
		{Name: "outbox", SoftRequires: []string{"queue"},
			Owns: []string{"outbox_events"}, Mount: contract.MountProtected},
		{Name: "dataops", Owns: []string{"import_jobs"}, Mount: contract.MountProtected},
		{Name: "search", Owns: []string{"search_documents"}, Mount: contract.MountProtected},
		{Name: "captcha", Mount: contract.MountPublic},
		{Name: "feature", Mount: contract.MountProtected},
		{Name: "uploadsec", Mount: contract.MountProtected},
		{Name: "breach", Mount: contract.MountProtected},
	}
}

// minimalShape 复刻 minimal profile 的形态（Task 5 放宽 auth.Requires 之后）：
// auth 的 tenant/mfa 是软依赖，因此 {user, access, auth} 是合法子集，tenant/mfa
// 缺席必须只降级、不报错。
func minimalShape() []contract.Descriptor {
	return []contract.Descriptor{
		{Name: "user", SoftRequires: []string{"access", "tenant"}},
		{Name: "access", Requires: []string{"user"}, SoftRequires: []string{"tenant"}},
		{Name: "auth", SoftRequires: []string{"tenant", "mfa", "captcha", "breach"}},
	}
}

func TestResolveEmptyMeansAll(t *testing.T) {
	got, err := Resolve(fixture(), nil)
	if err != nil {
		t.Fatalf("Resolve(nil) error: %v", err)
	}
	if len(got) != 18 {
		t.Fatalf("len = %d, want 18 (all)", len(got))
	}
}

func TestResolveUnknownCapability(t *testing.T) {
	_, err := Resolve(fixture(), []string{"nope"})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
	if !strings.Contains(err.Error(), "unknown capability") {
		t.Fatalf("error = %v, want unknown capability", err)
	}
}

func TestResolvePullsDependenciesAndKeepsOrder(t *testing.T) {
	// access 依赖 user，user 无依赖；顺序必须与清单一致（依赖在前）
	got, err := Resolve(fixture(), []string{"access"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "user" || got[1].Name != "access" {
		t.Fatalf("Resolve([access]) = %v, want [user access]", namesOf(got))
	}
}

func TestResolveClosureIsTransitive(t *testing.T) {
	// oauth -> auth -> access -> user；auth 的 tenant/mfa 是软依赖，不进闭包
	got, err := Resolve(fixture(), []string{"oauth"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	want := map[string]bool{"oauth": true, "auth": true, "user": true, "access": true}
	if len(got) != len(want) {
		t.Fatalf("Resolve([oauth]) = %v, want %d entries", namesOf(got), len(want))
	}
	for _, d := range got {
		if !want[d.Name] {
			t.Fatalf("unexpected capability %q in closure %v", d.Name, namesOf(got))
		}
	}
	wantOrder := []string{"user", "access", "auth", "oauth"}
	if gotOrder := namesOf(got); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("Resolve([oauth]) order = %v, want %v", gotOrder, wantOrder)
	}
}

func TestResolveIsSubsetOfAll(t *testing.T) {
	got, err := Resolve(fixture(), []string{"user"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "user" {
		t.Fatalf("Resolve([user]) = %v, want [user]", namesOf(got))
	}
}

func TestResolveRejectsUnknownDependency(t *testing.T) {
	_, err := Resolve([]contract.Descriptor{{Name: "broken", Requires: []string{"ghost"}}}, []string{"broken"})
	if err == nil || !strings.Contains(err.Error(), `requires unknown capability "ghost"`) {
		t.Fatalf("error = %v, want requires unknown capability", err)
	}
}

func TestResolveReturnsDeepCopyOfRequires(t *testing.T) {
	caps := []contract.Descriptor{{Name: "b"}, {Name: "a", Requires: []string{"b"}}}
	got, err := Resolve(caps, []string{"a"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	got[1].Requires[0] = "mutated"
	if again, err := Resolve(caps, []string{"a"}); err != nil || again[1].Requires[0] != "b" {
		t.Fatalf("Resolve() must not expose the registry's Requires backing array (err = %v)", err)
	}
}

// TestResolveReturnsDeepCopyOfSoftRequiresAndOwns 软依赖与自有表同样不得暴露清单底层数组。
func TestResolveReturnsDeepCopyOfSoftRequiresAndOwns(t *testing.T) {
	caps := []contract.Descriptor{{Name: "b"},
		{Name: "a", SoftRequires: []string{"b"}, Owns: []string{"t1"}}}
	got, err := Resolve(caps, []string{"a", "b"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	got[1].SoftRequires[0] = "mutated"
	got[1].Owns[0] = "mutated"
	again, err := Resolve(caps, []string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, "b", again[1].SoftRequires[0], "Resolve() must not expose the registry's SoftRequires backing array")
	assert.Equal(t, "t1", again[1].Owns[0], "Resolve() must not expose the registry's Owns backing array")
}

func TestResolveReturnsDeepCopyOfPermissions(t *testing.T) {
	caps := []contract.Descriptor{{Name: "a", Permissions: []contract.Permission{{Name: "p", Resource: "/r", Action: "GET"}}}}
	got, err := Resolve(caps, []string{"a"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	got[0].Permissions[0].Resource = "mutated"
	if again, err := Resolve(caps, []string{"a"}); err != nil || again[0].Permissions[0].Resource != "/r" {
		t.Fatalf("Resolve() must not expose the registry's Permissions backing array (err = %v)", err)
	}
}

func TestResolveReportsFirstDanglingDependencyInListOrder(t *testing.T) {
	// 两个坏依赖：错误必须稳定指向清单顺序里的第一个，而不是 map 遍历的随机一个
	caps := []contract.Descriptor{
		{Name: "a", Requires: []string{"ghost-a"}},
		{Name: "b", Requires: []string{"ghost-b"}},
	}
	for i := 0; i < 200; i++ {
		_, err := Resolve(caps, []string{"a", "b"})
		if err == nil || !strings.Contains(err.Error(), `"ghost-a"`) {
			t.Fatalf("run %d: error = %v, want the first dangling dependency in list order", i, err)
		}
	}
}

// TestResolveAllowsMissingSoftRequires minimal profile 子集：软依赖缺席必须解析成功，
// 且 Degraded 把它报告为降级项（user → tenant）。
func TestResolveAllowsMissingSoftRequires(t *testing.T) {
	got, err := Resolve(minimalShape(), []string{"user", "access", "auth"})
	require.NoError(t, err, "profile 子集允许软依赖缺席")
	require.Equal(t, []string{"user", "access", "auth"}, namesOf(got))

	degraded := Degraded(got)
	require.Len(t, degraded, 3)
	assert.Equal(t, Degradation{Capability: "user", Missing: []string{"tenant"}}, degraded[0])
	assert.Equal(t, Degradation{Capability: "access", Missing: []string{"tenant"}}, degraded[1])
	assert.Equal(t, Degradation{Capability: "auth", Missing: []string{"tenant", "mfa", "captcha", "breach"}}, degraded[2])
}

// TestResolveStillPullsHardRequiresWhenSoftMissing 软依赖可以被省略，硬依赖不行。
func TestResolveStillPullsHardRequiresWhenSoftMissing(t *testing.T) {
	got, err := Resolve(minimalShape(), []string{"access"})
	require.NoError(t, err)
	assert.Equal(t, []string{"user", "access"}, namesOf(got))
}

// TestValidateStructureIgnoresUnknownSoftDep 结构校验与集合无关：未知软依赖名不是
// 结构缺陷（catalog 的错别字检查由 ValidateDeclarations 用 known 完成）。
func TestValidateStructureIgnoresUnknownSoftDep(t *testing.T) {
	require.NoError(t, validateStructure([]contract.Descriptor{{Name: "a", SoftRequires: []string{"ghost"}}}))
}

// TestValidateStructureRejectsBadSoftRequires 结构缺陷（自引用/与硬依赖重叠/重复）必须报错。
func TestValidateStructureRejectsBadSoftRequires(t *testing.T) {
	cases := []struct {
		name string
		ds   []contract.Descriptor
	}{
		{"self soft dep", []contract.Descriptor{{Name: "a", SoftRequires: []string{"a"}}}},
		{"soft overlaps hard", []contract.Descriptor{
			{Name: "b"},
			{Name: "a", Requires: []string{"b"}, SoftRequires: []string{"b"}},
		}},
		{"duplicate soft dep", []contract.Descriptor{{Name: "b"}, {Name: "a", SoftRequires: []string{"b", "b"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, validateStructure(tc.ds))
		})
	}
}

// TestValidateDeclarationsChecksKnownAgainstList 软依赖必须命中 known（而不是 caps 本身）：
// 全量清单的错别字检查靠它，即使目标能力就在 caps 里，只要不在 known 内也算未知。
func TestValidateDeclarationsChecksKnownAgainstList(t *testing.T) {
	caps := []contract.Descriptor{{Name: "a", SoftRequires: []string{"b"}}, {Name: "b"}}

	require.NoError(t, ValidateDeclarations(caps, []string{"a", "b"}))
	require.Error(t, ValidateDeclarations(caps, []string{"a"}), "known 少一项时软依赖名不得命中")
	require.Error(t, ValidateDeclarations(caps, nil))
}

// TestValidateDeclarationsRejectsStructureDefects 完整校验包含结构校验。
func TestValidateDeclarationsRejectsStructureDefects(t *testing.T) {
	require.Error(t, ValidateDeclarations([]contract.Descriptor{{Name: "a", SoftRequires: []string{"a"}}}, []string{"a"}))
	require.Error(t, ValidateDeclarations([]contract.Descriptor{
		{Name: "b"},
		{Name: "a", Requires: []string{"b"}, SoftRequires: []string{"b"}},
	}, []string{"a", "b"}))
}

// TestDegradedListsMissingSoftDeps 只报告缺失的软依赖，硬依赖缺失由 Resolve 报错。
func TestDegradedListsMissingSoftDeps(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "user", SoftRequires: []string{"access", "tenant"}, Owns: []string{"users"}},
		{Name: "auth", SoftRequires: []string{"captcha", "breach"}, Owns: []string{"login_histories"}},
		{Name: "captcha"},
		{Name: "tenant"},
		{Name: "access", Requires: []string{"user"}},
	}
	got := Degraded(caps)
	require.Len(t, got, 1)
	assert.Equal(t, "auth", got[0].Capability)
	assert.Equal(t, []string{"breach"}, got[0].Missing)
}

// TestDegradedEmptyWhenAllSoftDepsPresent 依赖齐全时无降级项。
func TestDegradedEmptyWhenAllSoftDepsPresent(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "auth", SoftRequires: []string{"captcha"}},
		{Name: "captcha"},
	}
	assert.Empty(t, Degraded(caps))
}

func namesOf(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
