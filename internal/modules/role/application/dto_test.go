package application

import (
	"encoding/json"
	"testing"

	"jimu/internal/modules/role/domain"
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
}
