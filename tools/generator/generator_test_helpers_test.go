package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal/capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testrepo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
