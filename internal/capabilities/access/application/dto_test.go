package application

import (
	"encoding/json"
	"testing"

	"jimu/internal/capabilities/access/domain"
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
