package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorageUploadDownloadDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(filepath.Join(dir, "files"), "/files")
	require.NoError(t, err)

	key := "test/hello.txt"
	content := []byte("hello world")

	err = s.Upload(context.Background(), key, bytes.NewReader(content), int64(len(content)), "text/plain")
	require.NoError(t, err)

	ok, err := s.Exists(context.Background(), key)
	require.NoError(t, err)
	assert.True(t, ok)

	size, err := s.Size(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	rc, err := s.Download(context.Background(), key)
	require.NoError(t, err)
	defer rc.Close()
	got := make([]byte, len(content))
	_, err = rc.Read(got)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	require.NoError(t, s.Delete(context.Background(), key))
	ok, err = s.Exists(context.Background(), key)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestLocalStorageURL(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	assert.Equal(t, "/files/a/b.txt", s.URL("a/b.txt"))
}

func TestNewUnsupportedType(t *testing.T) {
	_, err := New(Config{Type: "ftp"})
	require.Error(t, err)
}

func TestNewS3RequiresBucket(t *testing.T) {
	_, err := New(Config{Type: StorageTypeS3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket is required")
}

func TestLocalStorageFilePersisted(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(dir, "/files")
	require.NoError(t, err)

	content := []byte("data")
	require.NoError(t, s.Upload(context.Background(), "x/y.txt", bytes.NewReader(content), int64(len(content)), "text/plain"))
	_, err = os.Stat(filepath.Join(dir, "x", "y.txt"))
	require.NoError(t, err)
}
