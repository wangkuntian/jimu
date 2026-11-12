package access

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func TestRoleSystemMetadataMigration(t *testing.T) {
	checked := 0
	for _, dialect := range []string{"mysql", "postgres"} {
		path := "migrations/" + dialect + "/009_role_system_metadata.sql"
		raw, err := fs.ReadFile(Descriptor.Migrations, path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		checked++
		sql := strings.ToLower(string(raw))
		for _, part := range []string{
			"add column code",
			"add column scope",
			"add column is_system",
			"idx_roles_code",
			"idx_roles_scope",
			"drop column",
		} {
			if !strings.Contains(sql, part) {
				t.Fatalf("%s missing %q", path, part)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no dialect migration found for 009_role_system_metadata.sql")
	}
}
