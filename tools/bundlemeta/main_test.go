package main

import (
	"path/filepath"
	"testing"

	"jimu/internal/templatesource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleMetaWritesManifest(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	output := filepath.Join(t.TempDir(), "template-bundle.json")

	err = run([]string{
		"--root", root,
		"--version", "v9.9.9",
		"--commit", "deadbeef",
		"--contract", "v0.1",
		"--output", output,
	})
	require.NoError(t, err)

	manifest, err := templatesource.LoadManifest(filepath.Dir(output))
	require.NoError(t, err)
	assert.Equal(t, "v9.9.9", manifest.BundleVersion)
	assert.Equal(t, "deadbeef", manifest.Commit)
	assert.Equal(t, "v0.1", manifest.ContractVersion)
	assert.NotEmpty(t, manifest.Files)
}
