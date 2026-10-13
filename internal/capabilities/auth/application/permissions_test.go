package application

import (
	"testing"

	"jimu/internal/kernel/access"
)

func TestEffectivePermissionsFiltersByRole(t *testing.T) {
	policies := []access.Policy{
		{Role: "admin", Resource: "/api/v1/tenants", Action: "GET"},
		{Role: "viewer", Resource: "/api/v1/audits", Action: "GET"},
		{Role: "admin", Resource: "/api/v1/admin/*", Action: "*"},
	}

	got := EffectivePermissions([]string{"admin"}, policies)

	want := []PermissionPoint{
		{Resource: "/api/v1/admin/*", Action: "*"},
		{Resource: "/api/v1/tenants", Action: "GET"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func TestEffectivePermissionsDeduplicatesAcrossRoles(t *testing.T) {
	policies := []access.Policy{
		{Role: "admin", Resource: "/api/v1/tenants", Action: "GET"},
		{Role: "operator", Resource: "/api/v1/tenants", Action: "GET"},
	}

	got := EffectivePermissions([]string{"admin", "operator"}, policies)

	if len(got) != 1 {
		t.Fatalf("got %#v, want exactly one point", got)
	}
}

func TestEffectivePermissionsSortsDeterministically(t *testing.T) {
	policies := []access.Policy{
		{Role: "admin", Resource: "/api/v1/tenants", Action: "POST"},
		{Role: "admin", Resource: "/api/v1/tenants", Action: "GET"},
		{Role: "admin", Resource: "/api/v1/audits", Action: "GET"},
	}

	got := EffectivePermissions([]string{"admin"}, policies)

	want := []PermissionPoint{
		{Resource: "/api/v1/audits", Action: "GET"},
		{Resource: "/api/v1/tenants", Action: "GET"},
		{Resource: "/api/v1/tenants", Action: "POST"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func TestEffectivePermissionsWithoutRolesIsEmptyNotNil(t *testing.T) {
	got := EffectivePermissions(nil, []access.Policy{
		{Role: "admin", Resource: "/api/v1/tenants", Action: "GET"},
	})

	if got == nil {
		t.Fatal("got nil, want an empty slice（JSON 必须是 [] 而不是 null）")
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestEffectivePermissionsIgnoresUnknownRoles(t *testing.T) {
	got := EffectivePermissions([]string{"ghost"}, []access.Policy{
		{Role: "admin", Resource: "/api/v1/tenants", Action: "GET"},
	})

	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}
