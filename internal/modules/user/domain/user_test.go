package domain

import "testing"

func TestUserTableName(t *testing.T) {
	if (User{}).TableName() != "users" {
		t.Fatal("users table name mismatch")
	}
}

func TestUserFields(t *testing.T) {
	u := User{ID: 1, Username: "admin", Password: "hash", Status: 1}
	if u.ID != 1 || u.Username != "admin" || u.Password != "hash" || u.Status != 1 {
		t.Fatal("user value mismatch")
	}
}
