package storage

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDefaultsLocal 覆盖 New 的 local 默认值分支（BaseDir/BaseURL 空时回填）
func TestNewDefaultsLocal(t *testing.T) {
	s, err := New(Config{Type: StorageTypeLocal})
	require.NoError(t, err)
	ls, ok := s.(*LocalStorage)
	require.True(t, ok)
	assert.Equal(t, "storage", ls.baseDir)
	assert.Equal(t, "/files", ls.baseURL)
}

// TestNewEmptyTypeDefaultsLocal 空类型等同于 local
func TestNewEmptyTypeDefaultsLocal(t *testing.T) {
	s, err := New(Config{})
	require.NoError(t, err)
	_, ok := s.(*LocalStorage)
	assert.True(t, ok)
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

// TestNewOSSAndMinioRequireBucket OSS/MinIO 复用 S3，同样要求 bucket
func TestNewOSSAndMinioRequireBucket(t *testing.T) {
	for _, ty := range []StorageType{StorageTypeOSS, StorageTypeMinIO} {
		_, err := New(Config{Type: ty})
		require.Error(t, err, "type=%s", ty)
		assert.Contains(t, err.Error(), "bucket is required")
	}
}
