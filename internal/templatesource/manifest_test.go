package templatesource

import (
	"reflect"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildManifestIncludesFilesAndExcludesManifest(t *testing.T) {
	bundle := fstest.MapFS{
		"go.mod":               &fstest.MapFile{Data: []byte("module jimu\n")},
		"internal/app.go":      &fstest.MapFile{Data: []byte("package internal\n")},
		"template-bundle.json": &fstest.MapFile{Data: []byte("{\"ignored\":true}\n")},
	}

	manifest, err := BuildManifest(bundle, []string{"go.mod", "internal"}, Info{
		CLIVersion:      "v9.9.9",
		BundleVersion:   "v9.9.9",
		Commit:          "deadbeef",
		ContractVersion: "v0.1",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, manifest.SchemaVersion)
	assert.Equal(t, "v9.9.9", manifest.BundleVersion)
	assert.Equal(t, "deadbeef", manifest.Commit)
	assert.Equal(t, "v0.1", manifest.ContractVersion)
	assert.Equal(t, []string{"go.mod", "internal/app.go"}, sortedKeys(manifest.Files))
	assert.NotContains(t, manifest.Files, "template-bundle.json")
}

func TestBuildManifestIsStable(t *testing.T) {
	bundle := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("a\n")},
		"b.txt": &fstest.MapFile{Data: []byte("b\n")},
	}
	info := Info{CLIVersion: "v1", BundleVersion: "v1", Commit: "abc", ContractVersion: "v0.1"}
	first, err := BuildManifest(bundle, []string{"a.txt", "b.txt"}, info)
	require.NoError(t, err)
	second, err := BuildManifest(bundle, []string{"a.txt", "b.txt"}, info)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(first, second))
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
