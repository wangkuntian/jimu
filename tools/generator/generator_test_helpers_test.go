package generator

import (
	"os"
	"path/filepath"
	"testing"
)

// withProtectedProvider returns a selectable capability expression that satisfies the
// preflight rule for every protected capability under test.
func withProtectedProvider(name string) string {
	switch name {
	case "auth", "apikey", "apidocs", "captcha", "oauth":
		return name
	default:
		return "apikey," + name
	}
}

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
