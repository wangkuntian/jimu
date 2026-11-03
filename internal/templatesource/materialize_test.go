package templatesource

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeWritesBundle(t *testing.T) {
	bundle := fstest.MapFS{
		"go.mod":         &fstest.MapFile{Data: []byte("module jimu\n"), Mode: 0o644},
		"internal/app":   &fstest.MapFile{Data: []byte("package app\n"), Mode: 0o644},
		"scripts/run.sh": &fstest.MapFile{Data: []byte("#!/bin/sh\n"), Mode: 0o644},
	}

	root, cleanup, err := Materialize(bundle)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanup() })

	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	assert.Equal(t, "module jimu\n", string(content))

	info, err := os.Stat(filepath.Join(root, "scripts", "run.sh"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestEmbeddedUsesDevFallbackMetadata(t *testing.T) {
	source, cleanup, err := Embedded(fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module jimu\n")},
	}, Info{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanup() })

	assert.NotEmpty(t, source.Framework.Version)
	assert.NotEmpty(t, source.Framework.Commit)
}
