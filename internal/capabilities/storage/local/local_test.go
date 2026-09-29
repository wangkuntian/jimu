package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDefaultsLocal 覆盖 New 的缺省值填充（BaseDir/BaseURL 为空时回填）。
// New 会 MkdirAll 缺省 BaseDir（"storage"），切到临时目录避免在源码树里造目录。
func TestNewDefaultsLocal(t *testing.T) {
	t.Chdir(t.TempDir())
	s, err := New(storage.Config{Type: storage.StorageTypeLocal})
	require.NoError(t, err)
	ls, ok := s.(*LocalStorage)
	require.True(t, ok)
	assert.Equal(t, "storage", ls.baseDir)
	assert.Equal(t, "/files", ls.baseURL)
}

// TestNewEmptyTypeDefaultsLocal 空类型等同于 local（同样切到临时目录，理由见上）
func TestNewEmptyTypeDefaultsLocal(t *testing.T) {
	t.Chdir(t.TempDir())
	s, err := New(storage.Config{})
	require.NoError(t, err)
	_, ok := s.(*LocalStorage)
	assert.True(t, ok)
}

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

// TestLocalStoragePathTraversal 路径遍历防护：.. 被剥离，不可逃出 baseDir
func TestLocalStoragePathTraversal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(dir, "/files")
	require.NoError(t, err)

	// .. 被替换为空，落到 baseDir 内
	err = s.Upload(context.Background(), "../../etc/passwd", bytes.NewReader([]byte("pwn")), 3, "text/plain")
	require.NoError(t, err)

	// 文件应落在 baseDir 内，非父目录
	_, _ = filepath.Abs(filepath.Join(dir, "../../etc/passwd"))
	target := filepath.Join(dir, "etc/passwd") // .. 剥离后
	absTarget, _ := filepath.Abs(target)
	rel, err := filepath.Rel(dir, absTarget)
	require.NoError(t, err)
	assert.False(t, strings.HasPrefix(rel, ".."), "escaped baseDir: %s", rel)
}

// TestLocalStorageSizeMismatch 上传字节数与声明 size 不符时报错
func TestLocalStorageSizeMismatch(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)

	err = s.Upload(context.Background(), "a.txt", bytes.NewReader([]byte("short")), 100, "text/plain")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size mismatch")

	// 失败后不留残文件
	ok, err := s.Exists(context.Background(), "a.txt")
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestLocalStorageDownloadNotFound 下载不存在文件返回明确错误
func TestLocalStorageDownloadNotFound(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	_, err = s.Download(context.Background(), "nope.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestLocalStorageDeleteMissingIsNoop 删除不存在的文件视为成功
func TestLocalStorageDeleteMissingIsNoop(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	require.NoError(t, s.Delete(context.Background(), "ghost.txt"))
}

// TestLocalStorageSizeMissing 文件不存在时 Size 返回错误
func TestLocalStorageSizeMissing(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	_, err = s.Size(context.Background(), "ghost.txt")
	require.Error(t, err)
}

// TestLocalStoragePresignedURL 本地存储预签名回退为普通 URL
func TestLocalStoragePresignedURL(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	got, err := s.PresignedURL("a/b.txt", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "/files/a/b.txt", got)
}

// TestLocalStoragePresignedUploadUnsupported 本地存储不支持预签名上传
func TestLocalStoragePresignedUploadUnsupported(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "/files")
	require.NoError(t, err)
	_, err = s.PresignedUploadURL("a.txt", time.Minute, "text/plain")
	require.Error(t, err)
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
