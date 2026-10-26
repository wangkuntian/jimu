package contract

import "testing"

func TestSystemRoleCatalog(t *testing.T) {
	platform := PlatformSystemRoles()
	tenant := TenantSystemRoles()
	if len(platform) != 2 || len(tenant) != 2 {
		t.Fatalf("system role counts = platform:%d tenant:%d, want 2/2", len(platform), len(tenant))
	}

	all := append(append([]SystemRoleSpec{}, platform...), tenant...)
	seen := make(map[string]int, len(all))
	for _, role := range all {
		seen[role.Code]++
	}
	for _, code := range []string{
		RoleCodePlatformSuperAdmin,
		RoleCodePlatformAuditor,
		RoleCodeTenantAdmin,
		RoleCodeTenantMember,
	} {
		if seen[code] != 1 {
			t.Fatalf("role code %q count = %d, want 1", code, seen[code])
		}
	}

	superAdmin := findSystemRole(t, platform, RoleCodePlatformSuperAdmin)
	if superAdmin.Scope != RoleScopePlatform || !superAdmin.AllPermissions {
		t.Fatalf("platform super admin = %#v", superAdmin)
	}
	auditor := findSystemRole(t, platform, RoleCodePlatformAuditor)
	if auditor.Scope != RoleScopePlatform || auditor.AllPermissions {
		t.Fatalf("platform auditor = %#v", auditor)
	}
	assertPermissions(t, auditor.Permissions, []ProvisioningPermission{
		{Resource: "/api/v1/tenants", Action: "GET"},
		{Resource: "/api/v1/users", Action: "GET"},
		{Resource: "/api/v1/users/*", Action: "GET"},
		{Resource: "/api/v1/roles", Action: "GET"},
		{Resource: "/api/v1/roles/*", Action: "GET"},
		{Resource: "/api/v1/roles/*/permissions", Action: "GET"},
		{Resource: "/api/v1/permissions", Action: "GET"},
		{Resource: "/api/v1/audits", Action: "GET"},
		{Resource: "/api/v1/audits/*", Action: "GET"},
		{Resource: "/api/v1/audits/export", Action: "GET"},
	})

	tenantAdmin := findSystemRole(t, tenant, RoleCodeTenantAdmin)
	if tenantAdmin.Scope != RoleScopeTenant || tenantAdmin.AllPermissions {
		t.Fatalf("tenant admin = %#v", tenantAdmin)
	}
	want := []ProvisioningPermission{
		{Resource: "/api/v1/users", Action: "GET"},
		{Resource: "/api/v1/users", Action: "POST"},
		{Resource: "/api/v1/users/*", Action: "GET"},
		{Resource: "/api/v1/users/*", Action: "PUT"},
		{Resource: "/api/v1/users/*", Action: "DELETE"},
		{Resource: "/api/v1/users/*/roles", Action: "POST"},
		{Resource: "/api/v1/roles/options", Action: "GET"},
		{Resource: "/api/v1/audits", Action: "GET"},
		{Resource: "/api/v1/audits/*", Action: "GET"},
	}
	assertPermissions(t, tenantAdmin.Permissions, want)

	tenantMember := findSystemRole(t, tenant, RoleCodeTenantMember)
	if tenantMember.Scope != RoleScopeTenant || len(tenantMember.Permissions) != 0 {
		t.Fatalf("tenant member = %#v", tenantMember)
	}
}

func TestPlatformOnlyPermissions(t *testing.T) {
	for _, permission := range []ProvisioningPermission{
		{Resource: "/api/v1/tenants", Action: "GET"},
		{Resource: "/api/v1/tenant-plans", Action: "POST"},
		{Resource: "/api/v1/permissions", Action: "GET"},
		{Resource: "/api/v1/roles", Action: "GET"},
		{Resource: "/api/v1/roles/*/permissions", Action: "GET"},
		{Resource: "/api/v1/admin/users", Action: "GET"},
	} {
		if !IsPlatformOnlyPermission(permission.Resource, permission.Action) {
			t.Fatalf("%s %s should be platform-only", permission.Action, permission.Resource)
		}
	}
	for _, permission := range []ProvisioningPermission{
		{Resource: "/api/v1/users", Action: "GET"},
		{Resource: "/api/v1/roles/options", Action: "GET"},
		{Resource: "/api/v1/audits", Action: "GET"},
	} {
		if IsPlatformOnlyPermission(permission.Resource, permission.Action) {
			t.Fatalf("%s %s should be tenant-assignable", permission.Action, permission.Resource)
		}
	}
}

func findSystemRole(t *testing.T, roles []SystemRoleSpec, code string) SystemRoleSpec {
	t.Helper()
	for _, role := range roles {
		if role.Code == code {
			return role
		}
	}
	t.Fatalf("missing system role %q", code)
	return SystemRoleSpec{}
}

func assertPermissions(t *testing.T, got, want []ProvisioningPermission) {
	t.Helper()
	counts := make(map[ProvisioningPermission]int, len(got))
	for _, item := range got {
		counts[item]++
	}
	if len(got) != len(want) {
		t.Fatalf("permissions = %#v, want %#v", got, want)
	}
	for _, item := range want {
		if counts[item] != 1 {
			t.Fatalf("permission %#v count = %d, want 1; got %#v", item, counts[item], got)
		}
	}
}
