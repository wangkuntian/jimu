package contract

import "testing"

// stubModule 只实现 Module 接口，用于验证 Describe 的回退行为。
type stubModule struct{ name string }

func (s stubModule) Name() string               { return s.name }
func (s stubModule) RegisterHTTP(r Router)      {}
func (s stubModule) RegisterJobs(j JobRegistry) {}
func (s stubModule) RegisterEvents(e EventBus)  {}

// describableStub 额外实现 Describable。
type describableStub struct {
	stubModule
	desc Descriptor
}

func (d describableStub) Descriptor() Descriptor { return d.desc }

func TestDescriptorNormalizedDefaultsToProtected(t *testing.T) {
	if got := (Descriptor{Name: "user"}).Normalized(); got != MountProtected {
		t.Fatalf("zero value mount = %q, want %q", got, MountProtected)
	}
	if got := (Descriptor{Name: "oauth", Mount: MountPublic}).Normalized(); got != MountPublic {
		t.Fatalf("explicit mount = %q, want %q", got, MountPublic)
	}
}

func TestDescribeFallsBackWhenNotDescribable(t *testing.T) {
	got := Describe(stubModule{name: "legacy"})
	if got.Name != "legacy" {
		t.Fatalf("name = %q, want %q", got.Name, "legacy")
	}
	if len(got.Requires) != 0 {
		t.Fatalf("requires = %v, want empty", got.Requires)
	}
	if got.Normalized() != MountProtected {
		t.Fatalf("mount = %q, want %q", got.Normalized(), MountProtected)
	}
}

func TestDescribeUsesDeclaredDescriptor(t *testing.T) {
	want := Descriptor{Name: "auth", Requires: []string{"user", "role", "tenant"}, Mount: MountSelfManaged}
	got := Describe(describableStub{stubModule: stubModule{name: "auth"}, desc: want})
	if got.Name != want.Name || got.Normalized() != want.Mount || len(got.Requires) != 3 {
		t.Fatalf("Describe() = %+v, want %+v", got, want)
	}
}
