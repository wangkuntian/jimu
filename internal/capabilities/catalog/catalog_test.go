package catalog

import (
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

// fixture 复刻 P0 八个能力的依赖形态（与真实清单拓扑一致）。
func fixture() []contract.Descriptor {
	return []contract.Descriptor{
		{Name: "user", Mount: contract.MountProtected},
		{Name: "role", Mount: contract.MountProtected},
		{Name: "permission", Requires: []string{"role"}, Mount: contract.MountProtected},
		{Name: "tenant", Mount: contract.MountProtected},
		{Name: "auth", Requires: []string{"user", "role", "tenant"}, Mount: contract.MountSelfManaged},
		{Name: "audit", Mount: contract.MountProtected},
		{Name: "admin", Requires: []string{"user", "audit"}, Mount: contract.MountProtected},
		{Name: "oauth", Requires: []string{"auth", "user"}, Mount: contract.MountPublic},
	}
}

func TestResolveEmptyMeansAll(t *testing.T) {
	withEntries(t, fixture()...)
	got, err := Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve(nil) error: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("len = %d, want 8 (all)", len(got))
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
	// permission 依赖 role，role 无依赖；顺序必须与清单一致（依赖在前）
	got, err := Resolve([]string{"permission"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "role" || got[1].Name != "permission" {
		t.Fatalf("Resolve([permission]) = %v, want [role permission]", namesOf(got))
	}
}

func TestResolveClosureIsTransitive(t *testing.T) {
	withEntries(t, fixture()...)
	// oauth -> auth -> user/role/tenant
	got, err := Resolve([]string{"oauth"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	want := map[string]bool{"oauth": true, "auth": true, "user": true, "role": true, "tenant": true}
	if len(got) != len(want) {
		t.Fatalf("Resolve([oauth]) = %v, want %d entries", namesOf(got), len(want))
	}
	for _, d := range got {
		if !want[d.Name] {
			t.Fatalf("unexpected capability %q in closure %v", d.Name, namesOf(got))
		}
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

// TestDescriptorsAreWellFormed 同时校验夹具与真实清单。Task 3 填齐清单后，
// 该用例对 catalog 的 8 个条目同样生效。
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
	}
}

func namesOf(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
