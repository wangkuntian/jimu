package application

import (
	"encoding/json"
	"testing"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
)

func TestRoleResponseUsesDTO(t *testing.T) {
	got := ToRoleResponse(domain.Role{ID: 1, Name: "admin", Description: "ops", Status: 1})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) || got.Name != "admin" {
		t.Fatalf("response = %#v json = %s", got, b)
	}
	if got.ID != "1" {
		t.Fatalf("role id = %q, want string id", got.ID)
	}
}

func TestPermissionResponseUsesStringID(t *testing.T) {
	got := ToPermissionResponse(domain.Permission{ID: 9007199254740993, Name: "read"})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "9007199254740993" || !json.Valid(b) {
		t.Fatalf("permission response = %#v json = %s", got, b)
	}
}

func TestRoleResponseCarriesSystemMetadata(t *testing.T) {
	system := ToRoleResponse(domain.Role{
		ID:       1,
		TenantID: 2,
		Name:     "租户管理员",
		Code:     contract.RoleCodeTenantAdmin,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
	})
	if system.Code != contract.RoleCodeTenantAdmin || system.Scope != contract.RoleScopeTenant || !system.IsSystem {
		t.Fatalf("system role response = %#v", system)
	}

	custom := ToRoleResponse(domain.Role{ID: 2, TenantID: 2, Name: "custom"})
	if custom.Code != "" || custom.Scope != contract.RoleScopeTenant || custom.IsSystem {
		t.Fatalf("custom role response = %#v", custom)
	}
}
