package frameworkmanifest

import (
	"slices"
	"strings"
	"testing"

	"jimu/internal/profiles/full"
)

func TestPresetNamesAreStable(t *testing.T) {
	want := []string{"minimal", "auth", "full"}
	if got := PresetNames(); !slices.Equal(got, want) {
		t.Fatalf("PresetNames() = %v, want %v", got, want)
	}
}

func TestMinimalPresetHasNoCapabilities(t *testing.T) {
	p, err := LookupPreset("minimal")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Capabilities) != 0 || len(p.Drivers) != 0 {
		t.Fatalf("minimal = %+v, want empty", p)
	}
	if p.Support != "official" {
		t.Fatalf("minimal support = %q, want official", p.Support)
	}
}

func TestAuthPresetSelectsAuthTriple(t *testing.T) {
	p, err := LookupPreset("auth")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"user", "access", "auth"}; !slices.Equal(p.Capabilities, want) {
		t.Fatalf("auth capabilities = %v, want %v", p.Capabilities, want)
	}
	if p.Support != "official" {
		t.Fatalf("auth support = %q, want official", p.Support)
	}
}

func TestFullPresetCoversFrameworkFull(t *testing.T) {
	p, err := LookupPreset("full")
	if err != nil {
		t.Fatal(err)
	}
	framework := full.Assembly().Capabilities
	if len(p.Capabilities) != len(framework) {
		t.Fatalf("full preset has %d capabilities, framework full has %d", len(p.Capabilities), len(framework))
	}
	selected := map[string]bool{}
	for _, name := range p.Capabilities {
		selected[name] = true
	}
	for _, item := range framework {
		if !selected[item.Descriptor.Name] {
			t.Fatalf("full preset missing %q", item.Descriptor.Name)
		}
	}
}

func TestFullPresetDriversMatchFrameworkFull(t *testing.T) {
	p, err := LookupPreset("full")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range full.Assembly().Capabilities {
		got := p.Drivers[item.Descriptor.Name]
		if !slices.Equal(got, item.Drivers) {
			t.Fatalf("driver %s = %v, want %v", item.Descriptor.Name, got, item.Drivers)
		}
	}
	if p.Support != "validated" {
		t.Fatalf("full support = %q, want validated", p.Support)
	}
}

func TestUnknownPresetListsAvailable(t *testing.T) {
	_, err := LookupPreset("ghost")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range PresetNames() {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error %q does not mention %q", err, name)
		}
	}
}
