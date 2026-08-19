package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS3StorageURLWithBaseURL 配置了 baseURL 时用它拼路径
func TestS3StorageURLWithBaseURL(t *testing.T) {
	s := &S3Storage{bucket: "b", baseURL: "https://cdn.example.com"}
	assert.Equal(t, "https://cdn.example.com/a/b.txt", s.URL("a/b.txt"))
}

// TestS3StorageURLWithoutBaseURL 无 baseURL 时走默认 S3 域名
func TestS3StorageURLWithoutBaseURL(t *testing.T) {
	s := &S3Storage{bucket: "b", baseURL: ""}
	got := s.URL("k")
	assert.Contains(t, got, "b.s3")
	assert.Contains(t, got, "/k")
}

// TestS3CompatibleStorageRequiresBucket 构造时 bucket 必填
func TestS3CompatibleStorageRequiresBucket(t *testing.T) {
	_, err := newS3CompatibleStorage(Config{Type: StorageTypeS3}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket is required")
}

// TestS3CompatibleStorageLoadsConfig 有 bucket + 静态凭证时构造成功（不发请求）
func TestS3CompatibleStorageLoadsConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		isS3 bool
	}{
		{"s3", Config{Bucket: "b", AccessKey: "ak", SecretKey: "sk", Region: "us-east-1"}, true},
		{"minio", Config{Bucket: "b", AccessKey: "ak", SecretKey: "sk", Endpoint: "http://localhost:9000"}, false},
		{"oss", Config{Bucket: "b", AccessKey: "ak", SecretKey: "sk", Endpoint: "http://oss.example.com"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := newS3CompatibleStorage(tc.cfg, tc.isS3)
			require.NoError(t, err)
			impl, ok := s.(*S3Storage)
			require.True(t, ok)
			assert.Equal(t, "b", impl.bucket)
			assert.NotNil(t, impl.client)
		})
	}
}
