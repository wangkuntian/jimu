package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelfCheckRunsTestsWhenProjectHasNoCapabilityGate(t *testing.T) {
	dir := t.TempDir()
	writeSelfcheckFile(t, dir, "go.mod", "module example.com/basic\n\ngo 1.26.10\n")
	writeSelfcheckFile(t, dir, "basic.go", "package basic\n\nfunc Value() int { return 1 }\n")
	writeSelfcheckFile(t, dir, "basic_test.go", `package basic

import "testing"

func TestValue(t *testing.T) {
	if Value() != 1 {
		t.Fatal("unexpected value")
	}
}
`)

	require.NoError(t, SelfCheck(dir))
}

func writeSelfcheckFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
