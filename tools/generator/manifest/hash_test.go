package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashFilesReturnsContentSHA256(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644))

	got, err := HashFiles(root, []string{"b.txt", "a.txt"})
	require.NoError(t, err)
	assert.Equal(t, "sha256:"+digest("a\n"), got["a.txt"])
	assert.Equal(t, "sha256:"+digest("b\n"), got["b.txt"])

	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("changed\n"), 0o644))
	changed, err := HashFiles(root, []string{"a.txt"})
	require.NoError(t, err)
	assert.NotEqual(t, got["a.txt"], changed["a.txt"])
}

func TestHashFilesRejectsMissingAndUnsafePaths(t *testing.T) {
	root := t.TempDir()
	for _, files := range [][]string{{"missing.txt"}, {"../outside.txt"}} {
		_, err := HashFiles(root, files)
		require.Error(t, err)
	}
}

func TestValidateCompleteRequiresEveryGeneratedFileHash(t *testing.T) {
	doc := validDocument()
	doc.GeneratedFiles = []string{"a.txt", "b.txt"}
	doc.GeneratedFileSHA256 = map[string]string{
		"a.txt": "sha256:" + strings.Repeat("a", 64),
		"b.txt": "sha256:" + strings.Repeat("b", 64),
	}
	require.NoError(t, ValidateComplete(doc))

	delete(doc.GeneratedFileSHA256, "b.txt")
	require.ErrorContains(t, ValidateComplete(doc), "missing generated file hash")

	doc.GeneratedFileSHA256["b.txt"] = "sha256:" + strings.Repeat("b", 64)
	doc.GeneratedFileSHA256["extra.txt"] = "sha256:" + strings.Repeat("c", 64)
	require.ErrorContains(t, ValidateComplete(doc), "not listed")
}

func TestDigestIncludesGeneratedFileHashes(t *testing.T) {
	doc := validDocument()
	doc.GeneratedFiles = []string{"a.txt"}
	doc.GeneratedFileSHA256 = map[string]string{"a.txt": "sha256:" + strings.Repeat("a", 64)}
	first, err := Digest(doc)
	require.NoError(t, err)

	doc.GeneratedFileSHA256["a.txt"] = "sha256:" + strings.Repeat("b", 64)
	second, err := Digest(doc)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
}

func TestValidateCompleteRequiresProvenance(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*Document)
		want string
	}{
		{"framework version", func(doc *Document) { doc.Framework.Version = "" }, "framework.version"},
		{"framework commit", func(doc *Document) { doc.Framework.Commit = "" }, "framework.commit"},
		{"template version", func(doc *Document) { doc.Selection.Template = "basic-api" }, "template_version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := validDocument()
			tc.edit(&doc)
			require.ErrorContains(t, ValidateComplete(doc), tc.want)
		})
	}
}

func TestWriteRejectsIncompleteManifest(t *testing.T) {
	doc := validDocument()
	doc.GeneratedFiles = []string{"go.mod"}
	path := filepath.Join(t.TempDir(), ".jimu", "manifest.json")
	require.ErrorContains(t, Write(path, doc), "missing generated file hash")
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
