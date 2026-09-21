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
	target := filepath.Join(root, "internal/capabilities/product/domain/entity.go")
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
	// 存量编号沿用原全局值：能力目录已有 001/006 → 新迁移取 007
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
	// writeMigration 硬编码 order_item 目录；本用例生成 product，改为直接在该能力目录预置基线迁移
	baseDir := filepath.Join(root, "internal/capabilities/product/migrations/mysql")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "001_base.sql"), []byte("-- migration\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	if _, err := os.Stat(filepath.Join(root, "internal", "capabilities", "product", "migrations", "mysql", "001_base.sql")); err != nil {
		t.Fatalf("base migration changed: %v", err)
	}
	// 生成器自建的目录必须回滚；预置的 mysql/ 基线目录允许保留
	if _, err := os.Stat(filepath.Join(root, "internal", "capabilities", "product", "module.go")); !os.IsNotExist(err) {
		t.Fatalf("module file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "capabilities", "product", "migrations", "postgres")); !os.IsNotExist(err) {
		t.Fatalf("postgres migration dir still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "capabilities", "product", "migrations", "mysql", "002_create_products.sql")); !os.IsNotExist(err) {
		t.Fatalf("migration still exists: %v", err)
	}
}

func TestGenerateModuleStartsCapabilityNumberingAt001(t *testing.T) {
	root := newTestRepository(t)
	// 新能力目录为空：能力内自行编号从这里开始生效（从 001 起），不受其他能力迁移影响
	if err := GenerateModuleAt(root, "product"); err != nil {
		t.Fatal(err)
	}
	mig := filepath.Join(root, "internal", "capabilities", "product", "migrations", "mysql", "001_create_products.sql")
	if _, err := os.Stat(mig); err != nil {
		t.Fatalf("missing %s: %v", mig, err)
	}
}

func newTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "internal/capabilities"),
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
	entries, err := os.ReadDir(filepath.Join(root, "internal/capabilities"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("generated modules = %d", len(entries))
	}
}

func writeMigration(t *testing.T, root, name string) {
	t.Helper()
	// ponytail: 硬编码 order_item 目录（两个调用方都生成 order_item/product），
	// 需要按能力写入时给 writeMigration 加 cap 参数即可。
	dir := filepath.Join(root, "internal/capabilities/order_item/migrations/mysql")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("-- migration\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requiredFiles(name, version string) []string {
	capRoot := filepath.Join("internal", "capabilities", name)
	return []string{
		filepath.Join(capRoot, "module.go"),
		filepath.Join(capRoot, "domain", "entity.go"),
		filepath.Join(capRoot, "domain", "repository.go"),
		filepath.Join(capRoot, "application", "dto.go"),
		filepath.Join(capRoot, "application", "service.go"),
		filepath.Join(capRoot, "application", "service_test.go"),
		filepath.Join(capRoot, "infrastructure", "mysql_repository.go"),
		filepath.Join(capRoot, "interfaces", "handler.go"),
		filepath.Join(capRoot, "interfaces", "handler_test.go"),
		filepath.Join(capRoot, "interfaces", "router.go"),
		filepath.Join(capRoot, "migrations", "mysql", version+"_create_order_items.sql"),
		filepath.Join(capRoot, "migrations", "postgres", version+"_create_order_items.sql"),
		filepath.Join(capRoot, "migrations", "postgres", ".gitkeep"),
	}
}
