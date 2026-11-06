package generator

import (
	"path/filepath"
	"testing"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectUsesProvidedSourceProvenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	provenance := &manifest.Framework{
		Module:  "jimu",
		Version: "v9.9.9",
		Commit:  "deadbeef",
		Dirty:   false,
	}

	_, err := NewProject(NewOptions{
		Dir:         dir,
		Template:    "minimal",
		Module:      "example.com/proj",
		SourceRoot:  FrameworkRoot(),
		Framework:   provenance,
		NoTidy:      true,
		NoSelfCheck: true,
	})
	require.NoError(t, err)

	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, provenance.Module, doc.Framework.Module)
	assert.Equal(t, provenance.Version, doc.Framework.Version)
	assert.Equal(t, provenance.Commit, doc.Framework.Commit)
	assert.Equal(t, provenance.Dirty, doc.Framework.Dirty)
}
