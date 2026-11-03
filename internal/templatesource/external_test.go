package templatesource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalValidatesVersionCommitAndHashes(t *testing.T) {
	root := writeExternalSource(t)
	info := externalInfo()

	source, err := externalWithRoots(root, []string{"go.mod", "internal"}, info)
	require.NoError(t, err)
	assert.Equal(t, root, source.Root)
	assert.Equal(t, "v9.9.9", source.Framework.Version)
	assert.Equal(t, "deadbeef", source.Framework.Commit)
}

func TestExternalRejectsMissingManifest(t *testing.T) {
	root := t.TempDir()
	_, err := externalWithRoots(root, []string{"go.mod"}, externalInfo())
	require.ErrorContains(t, err, "template-bundle.json")
}

func TestExternalRejectsVersionMismatch(t *testing.T) {
	root := writeExternalSource(t)
	info := externalInfo()
	info.BundleVersion = "v0.0.1"
	_, err := externalWithRoots(root, []string{"go.mod", "internal"}, info)
	require.ErrorContains(t, err, "version")
}

func TestExternalRejectsCommitMismatch(t *testing.T) {
	root := writeExternalSource(t)
	info := externalInfo()
	info.Commit = "cafebabe"
	_, err := externalWithRoots(root, []string{"go.mod", "internal"}, info)
	require.ErrorContains(t, err, "commit")
}

func TestExternalRejectsHashMismatch(t *testing.T) {
	root := writeExternalSource(t)
	writeTestFile(t, root, "internal/app.go", "package changed\n")
	_, err := externalWithRoots(root, []string{"go.mod", "internal"}, externalInfo())
	require.ErrorContains(t, err, "hash")
}

func TestExternalRejectsMissingAndExtraFiles(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		root := writeExternalSource(t)
		require.NoError(t, os.Remove(filepath.Join(root, "internal", "app.go")))
		_, err := externalWithRoots(root, []string{"go.mod", "internal"}, externalInfo())
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "missing") || strings.Contains(err.Error(), "hash"))
	})

	t.Run("extra", func(t *testing.T) {
		root := writeExternalSource(t)
		writeTestFile(t, root, "internal/extra.go", "package extra\n")
		_, err := externalWithRoots(root, []string{"go.mod", "internal"}, externalInfo())
		require.ErrorContains(t, err, "file set")
	})
}

func externalInfo() Info {
	return Info{
		CLIVersion:      "v9.9.9",
		BundleVersion:   "v9.9.9",
		Commit:          "deadbeef",
		ContractVersion: "v0.1",
	}
}

func writeExternalSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module jimu\n")
	writeTestFile(t, root, "internal/app.go", "package internal\n")
	manifest, err := BuildManifest(os.DirFS(root), []string{"go.mod", "internal"}, externalInfo())
	require.NoError(t, err)
	require.NoError(t, WriteManifest(root, manifest))
	return root
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
