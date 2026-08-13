package domain

import "testing"

func TestTableNames(t *testing.T) {
	if (Role{}).TableName() != "roles" {
		t.Fatal("roles table name mismatch")
	}
	if (Permission{}).TableName() != "permissions" {
		t.Fatal("permissions table name mismatch")
	}
}

func TestEntityValues(t *testing.T) {
	r := Role{ID: 1, Name: "admin", Status: 1}
	if r.ID != 1 || r.Name != "admin" || r.Status != 1 {
		t.Fatal("role value mismatch")
	}
	p := Permission{ID: 1, Name: "create_user", Resource: "user", Action: "create"}
	if p.ID != 1 || p.Resource != "user" || p.Action != "create" {
		t.Fatal("permission value mismatch")
	}
}
