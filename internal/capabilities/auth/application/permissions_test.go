package application

import (
	"testing"

	"jimu/internal/kernel/access"
)

func TestEffectivePermissionsFiltersByRole(t *testing.T) {
	policies := []access.Policy{
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
		{Role: access.RoleSubject(2), Resource: "/api/v1/audits", Action: "GET"},
		{Role: access.RoleSubject(1), Resource: "/api/v1/admin/*", Action: "*"},
	}

	got := EffectivePermissions([]string{access.RoleSubject(1)}, policies)

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
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
		{Role: access.RoleSubject(2), Resource: "/api/v1/tenants", Action: "GET"},
	}

	got := EffectivePermissions([]string{access.RoleSubject(1), access.RoleSubject(2)}, policies)

	want := PermissionPoint{Resource: "/api/v1/tenants", Action: "GET"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %#v, want exactly [%#v]", got, want)
	}
}

func TestEffectivePermissionsSortsDeterministically(t *testing.T) {
	policies := []access.Policy{
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "POST"},
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
		{Role: access.RoleSubject(1), Resource: "/api/v1/audits", Action: "GET"},
	}

	got := EffectivePermissions([]string{access.RoleSubject(1)}, policies)

	want := []PermissionPoint{
		{Resource: "/api/v1/audits", Action: "GET"},
		{Resource: "/api/v1/tenants", Action: "GET"},
		{Resource: "/api/v1/tenants", Action: "POST"},
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

func TestEffectivePermissionsWithoutRolesIsEmptyNotNil(t *testing.T) {
	got := EffectivePermissions(nil, []access.Policy{
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
	})

	if got == nil {
		t.Fatal("got nil, want an empty slice（JSON 必须是 [] 而不是 null）")
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestEffectivePermissionsIgnoresUnknownRoles(t *testing.T) {
	got := EffectivePermissions([]string{access.RoleSubject(99)}, []access.Policy{
		{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
	})

	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}
