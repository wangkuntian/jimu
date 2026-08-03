package generator

import (
	"bytes"
	"errors"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModuleRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "Product", "order-item", "order__item", "order_", "type"} {
		t.Run(name, func(t *testing.T) {
			root := newTestRepository(t)
			if err := GenerateModuleAt(root, name); err == nil {
				t.Fatal("expected validation error")
			}
			assertNoGeneratedFiles(t, root)
		})
	}
}

func TestGenerateModuleDoesNotOverwriteExistingTarget(t *testing.T) {
	root := newTestRepository(t)
	target := filepath.Join(root, "internal/modules/product/domain/entity.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateModuleAt(root, "product"); err == nil {
		t.Fatal("expected target conflict")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing target changed: %q, %v", got, err)
	}
}

func TestGenerateModuleCreatesCompleteCRUD(t *testing.T) {
	root := newTestRepository(t)
	writeMigration(t, root, "001_create_users.sql")
	writeMigration(t, root, "006_create_roles.sql")

	if err := GenerateModuleAt(root, "order_item"); err != nil {
		t.Fatal(err)
	}
	for _, rel := range requiredFiles("order_item", "007") {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte("TODO")) || bytes.Contains(b, []byte("gin.Error{}")) {
			t.Errorf("unfinished marker in %s", path)
		}
		if strings.HasSuffix(path, ".go") {
			if _, err := format.Source(b); err != nil {
				t.Errorf("unformatted go file %s: %v", path, err)
			}
		}
		return nil
	})
}

func TestGenerateModuleRollsBackWriteFailure(t *testing.T) {
	root := newTestRepository(t)
	writeMigration(t, root, "001_base.sql")
	originalWriteFile := writeFile
	t.Cleanup(func() { writeFile = originalWriteFile })
	writes := 0
	writeFile = func(name string, data []byte, perm os.FileMode) error {
		writes++
		if writes == 3 {
			return errors.New("disk full")
		}
		return originalWriteFile(name, data, perm)
	}

	if err := GenerateModuleAt(root, "product"); err == nil {
		t.Fatal("expected write failure")
	}
	if _, err := os.Stat(filepath.Join(root, "migrations", "001_base.sql")); err != nil {
		t.Fatalf("base migration changed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "modules", "product")); !os.IsNotExist(err) {
		t.Fatalf("module directory still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "migrations", "002_create_products.sql")); !os.IsNotExist(err) {
		t.Fatalf("migration still exists: %v", err)
	}
}

func newTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "internal/modules"),
		filepath.Join(root, "migrations"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testrepo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertNoGeneratedFiles(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "internal/modules"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("generated modules = %d", len(entries))
	}
}

func writeMigration(t *testing.T, root, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "migrations", name), []byte("-- migration\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requiredFiles(name, version string) []string {
	return []string{
		filepath.Join("internal", "modules", name, "module.go"),
		filepath.Join("internal", "modules", name, "domain", "entity.go"),
		filepath.Join("internal", "modules", name, "domain", "repository.go"),
		filepath.Join("internal", "modules", name, "application", "dto.go"),
		filepath.Join("internal", "modules", name, "application", "service.go"),
		filepath.Join("internal", "modules", name, "application", "service_test.go"),
		filepath.Join("internal", "modules", name, "infrastructure", "mysql_repository.go"),
		filepath.Join("internal", "modules", name, "interfaces", "handler.go"),
		filepath.Join("internal", "modules", name, "interfaces", "handler_test.go"),
		filepath.Join("internal", "modules", name, "interfaces", "router.go"),
		filepath.Join("migrations", version+"_create_order_items.sql"),
	}
}
