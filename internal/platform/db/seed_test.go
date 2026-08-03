package db

import "testing"

func TestBasePermissionsCoverBusinessRoutes(t *testing.T) {
	required := []struct {
		resource string
		action   string
	}{
		{"/api/v1/users", "GET"},
		{"/api/v1/users", "POST"},
		{"/api/v1/users/*", "GET"},
		{"/api/v1/users/*", "PUT"},
		{"/api/v1/users/*", "DELETE"},
		{"/api/v1/roles", "GET"},
		{"/api/v1/roles", "POST"},
		{"/api/v1/roles/*", "GET"},
		{"/api/v1/roles/*", "PUT"},
		{"/api/v1/roles/*", "DELETE"},
		{"/api/v1/roles/*/permissions", "POST"},
		{"/api/v1/permissions", "GET"},
		{"/api/v1/permissions", "POST"},
		{"/api/v1/permissions/*", "GET"},
		{"/api/v1/permissions/*", "PUT"},
		{"/api/v1/permissions/*", "DELETE"},
		{"/api/v1/audits", "GET"},
		{"/api/v1/audits/*", "GET"},
	}
	got := make(map[string]bool)
	for _, permission := range basePermissions() {
		got[permission.Resource+" "+permission.Action] = true
	}
	for _, item := range required {
		if !got[item.resource+" "+item.action] {
			t.Fatalf("missing permission %s %s", item.action, item.resource)
		}
	}
}
