package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockS3Storage 起 httptest mock S3（path style），返回连向它的 S3Storage
func newMockS3Storage(t *testing.T) *S3Storage {
	t.Helper()
	var mu sync.Mutex
	objects := map[string][]byte{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// path style: /{bucket}/{key...}
		key := strings.TrimPrefix(r.URL.Path, "/b/")
		if key == r.URL.Path || key == "" {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			objects[key] = b
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			mu.Lock()
			b, ok := objects[key]
			mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(b)
		case http.MethodHead:
			mu.Lock()
			b, ok := objects[key]
			mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(b)))
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			mu.Lock()
			delete(objects, key)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)

	cfg := Config{
		Type:      StorageTypeMinIO,
		Bucket:    "b",
		AccessKey: "ak",
		SecretKey: "sk",
		Region:    "us-east-1",
		Endpoint:  srv.URL,
	}
	s, err := newS3CompatibleStorage(cfg, false)
	require.NoError(t, err)
	return s.(*S3Storage)
}

// TestS3UploadDownloadRoundTrip 上传→存在→大小→下载→删除 全链路
func TestS3UploadDownloadRoundTrip(t *testing.T) {
	s := newMockS3Storage(t)
	ctx := context.Background()
	content := []byte("hello s3")

	require.NoError(t, s.Upload(ctx, "dir/a.txt", bytes.NewReader(content), int64(len(content)), "text/plain"))

	ok, err := s.Exists(ctx, "dir/a.txt")
	require.NoError(t, err)
	assert.True(t, ok)

	size, err := s.Size(ctx, "dir/a.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	rc, err := s.Download(ctx, "dir/a.txt")
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	rc.Close()
	require.NoError(t, err)
	assert.Equal(t, content, got)

	require.NoError(t, s.Delete(ctx, "dir/a.txt"))
	ok, err = s.Exists(ctx, "dir/a.txt")
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestS3ExistsAndSizeMissing 不存在的对象：Exists 返回 false，Size 报错
func TestS3ExistsAndSizeMissing(t *testing.T) {
	s := newMockS3Storage(t)
	ctx := context.Background()

	ok, err := s.Exists(ctx, "nope.txt")
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = s.Size(ctx, "nope.txt")
	require.Error(t, err)

	_, err = s.Download(ctx, "nope.txt")
	require.Error(t, err)
}

// TestS3DeleteMissingIsNoop 删除不存在的对象不报错
func TestS3DeleteMissingIsNoop(t *testing.T) {
	s := newMockS3Storage(t)
	require.NoError(t, s.Delete(context.Background(), "ghost.txt"))
}
