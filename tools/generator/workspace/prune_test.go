package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writePruneFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.26\n",
		"internal/keep/keep.go": `package keep
`,
		"internal/keep/keep_test.go": `package keep
`,
		"internal/drop/drop.go": `package drop
`,
		"internal/runtime/runtime.go": `package runtime
`,
		"internal/testonly/testonly.go": `package testonly

import _ "example.com/app/internal/runtime"
`,
		"internal/testonly/testonly_test.go": `package testonly
`,
		"internal/testutil/testutil.go": `package testutil

import _ "example.com/app/internal/platform/db"
`,
		"internal/testutil/testutil_test.go": `package testutil
`,
		"internal/platform/db/db.go": `package db
`,
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func TestPrunePackagesDropsRuntimeUnreachablePackage(t *testing.T) {
	dir := writePruneFixture(t)
	result, err := PrunePackages(dir, PruneOptions{
		Module:  "example.com/app",
		Tests:   "selected",
		Runtime: map[string]bool{"example.com/app/internal/keep": true},
	})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "internal/keep/keep.go"))
	require.FileExists(t, filepath.Join(dir, "internal/keep/keep_test.go"))
	require.NoFileExists(t, filepath.Join(dir, "internal/drop/drop.go"))
	require.Contains(t, result.Removed, "internal/drop/drop.go")
}

func TestPrunePackagesTestsNoneRemovesAllTestFiles(t *testing.T) {
	dir := writePruneFixture(t)
	_, err := PrunePackages(dir, PruneOptions{Module: "example.com/app", Tests: "none", Runtime: map[string]bool{}})
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(dir, "internal/keep/keep_test.go"))
}

func TestPrunePackagesKeepsSafeTestOnlyPackage(t *testing.T) {
	dir := writePruneFixture(t)
	_, err := PrunePackages(dir, PruneOptions{
		Module:  "example.com/app",
		Tests:   "all",
		Runtime: map[string]bool{"example.com/app/internal/runtime": true},
		TestDeps: map[string]bool{
			"example.com/app/internal/testonly": true,
			"example.com/app/internal/testutil": true,
		},
	})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "internal/testonly/testonly.go"))
	require.FileExists(t, filepath.Join(dir, "internal/testonly/testonly_test.go"))
	require.NoFileExists(t, filepath.Join(dir, "internal/testutil/testutil.go"))
	require.NoFileExists(t, filepath.Join(dir, "internal/testutil/testutil_test.go"))
}
