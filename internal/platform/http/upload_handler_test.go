package http

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"jimu/internal/platform/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStorage 内存实现，覆盖 storage.Storage 全接口
type fakeStorage struct {
	uploaded map[string]string // key → contentType
	deleted  []string
	err      error
}

func newFakeStorage() *fakeStorage { return &fakeStorage{uploaded: map[string]string{}} }

func (f *fakeStorage) Upload(_ context.Context, key string, _ io.Reader, _ int64, contentType string) error {
	if f.err != nil {
		return f.err
	}
	f.uploaded[key] = contentType
	return nil
}
func (f *fakeStorage) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (f *fakeStorage) Delete(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, key)
	return nil
}
func (f *fakeStorage) Exists(context.Context, string) (bool, error) { return false, nil }
func (f *fakeStorage) Size(context.Context, string) (int64, error)  { return 0, nil }
func (f *fakeStorage) URL(key string) string                        { return "http://example.com/" + key }
func (f *fakeStorage) PresignedURL(string, time.Duration) (string, error) {
	return "", nil
}
func (f *fakeStorage) PresignedUploadURL(string, time.Duration, string) (string, error) {
	return "", nil
}

var _ storage.Storage = (*fakeStorage)(nil)

func uploadEngine(h *UploadHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", h.HandleUpload())
	r.POST("/upload-multi", h.HandleUploadMultiple("files", 2))
	r.DELETE("/delete", h.HandleDelete())
	return r
}

// multipartFileRequestWithType 构造带指定 part Content-Type 的单文件 multipart 请求
func multipartFileRequestWithType(t *testing.T, field, filename, partType, content string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, filename))
	if partType != "" {
		h.Set("Content-Type", partType)
	}
	fw, err := mw.CreatePart(h)
	require.NoError(t, err)
	_, err = fw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestNewUploadHandlerDefaults(t *testing.T) {
	h := NewUploadHandler(UploadConfig{Storage: newFakeStorage()})
	require.NotNil(t, h)
	assert.Equal(t, int64(10*1024*1024), h.maxSize)
	assert.Equal(t, "uploads", h.basePrefix)
}

func TestNewUploadHandlerKeepsCustomValues(t *testing.T) {
	h := NewUploadHandler(UploadConfig{Storage: newFakeStorage(), MaxSize: 100, AllowTypes: "image/", BasePrefix: "avatars"})
	assert.Equal(t, int64(100), h.maxSize)
	assert.Equal(t, "image/", h.allowTypes)
	assert.Equal(t, "avatars", h.basePrefix)
}

func TestIsAllowedType(t *testing.T) {
	assert.True(t, isAllowedType("image/png", "image/,application/pdf"))
	assert.True(t, isAllowedType("application/pdf", "image/,application/pdf"))
	assert.True(t, isAllowedType("anything", ""))
	assert.False(t, isAllowedType("text/plain", "image/"))
	assert.False(t, isAllowedType("image/png", "image/jpeg"))
	assert.False(t, isAllowedType("", "image/"))
}

func TestReadAll(t *testing.T) {
	b, err := readAll(strings.NewReader("hello"), 10)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))

	_, err = readAll(strings.NewReader("hello world"), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max size")
}

func TestHandleUploadMissingFile(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/upload", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadSuccess(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st, BasePrefix: "uploads"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", h.HandleUpload())

	req := multipartFileRequestWithType(t, "file", "hello.txt", "text/plain", "hello world")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, st.uploaded, 1)
	assert.Contains(t, w.Body.String(), `"code":0`)
	// content_type 经 magic-byte 嗅探，文本附加 charset；校验真实类型前缀
	assert.Contains(t, w.Body.String(), `"content_type":"text/plain`)
	// key 使用默认前缀 + 扩展名
	for key, ct := range st.uploaded {
		assert.True(t, strings.HasPrefix(ct, "text/plain"), "ct=%s", ct)
		assert.True(t, strings.HasPrefix(key, "uploads/"), "key=%s", key)
		assert.True(t, strings.HasSuffix(key, ".txt"), "key=%s", key)
	}
}

func TestHandleUploadCustomField(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", h.HandleUpload("document"))

	req := multipartFileRequestWithType(t, "document", "a.pdf", "application/pdf", "pdf-data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, st.uploaded, 1)
}

func TestHandleUploadTooLarge(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st, MaxSize: 4})
	r := uploadEngine(h)
	req := multipartFileRequestWithType(t, "file", "big.txt", "text/plain", "hello")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadEmptyFile(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	req := multipartFileRequestWithType(t, "file", "empty.txt", "text/plain", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadTypeNotAllowed(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st, AllowTypes: "image/"})
	r := uploadEngine(h)
	req := multipartFileRequestWithType(t, "file", "note.txt", "text/plain", "data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadStorageError(t *testing.T) {
	st := newFakeStorage()
	st.err = errors.New("s3 down")
	h := NewUploadHandler(UploadConfig{Storage: st})
	r := uploadEngine(h)
	req := multipartFileRequestWithType(t, "file", "a.txt", "text/plain", "data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleUploadMultipleSuccess(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload-multi", h.HandleUploadMultiple("files", 3))

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	for _, f := range []struct{ name, content string }{{"a.txt", "aaa"}, {"b.txt", "bbb"}} {
		fw, err := mw.CreateFormFile("files", f.name)
		require.NoError(t, err)
		_, _ = fw.Write([]byte(f.content))
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/upload-multi", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, st.uploaded, 2)
	assert.Contains(t, w.Body.String(), `"code":0`)
}

func TestHandleUploadMultipleNoFiles(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/upload-multi", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadMultipleTooManyFiles(t *testing.T) {
	st := newFakeStorage()
	h := NewUploadHandler(UploadConfig{Storage: st})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload-multi", h.HandleUploadMultiple("files", 2))

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	for _, n := range []string{"a", "b", "c"} {
		fw, err := mw.CreateFormFile("files", n+".txt")
		require.NoError(t, err)
		_, _ = fw.Write([]byte("x"))
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/upload-multi", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleUploadMultipleInvalidForm(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	req := httptest.NewRequest(http.MethodPost, "/upload-multi", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded)
}

func TestHandleDeleteSuccess(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/delete?key=uploads/1.jpg", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"uploads/1.jpg"}, st.deleted)
	assert.Contains(t, w.Body.String(), `"code":0`)
}

func TestHandleDeleteMissingKey(t *testing.T) {
	st := newFakeStorage()
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/delete", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDeleteStorageError(t *testing.T) {
	st := newFakeStorage()
	st.err = errors.New("del failed")
	r := uploadEngine(NewUploadHandler(UploadConfig{Storage: st}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/delete?key=k", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// fakeScanner 内存假扫描器，按预设结果返回
type fakeScanner struct {
	clean bool
	err   error
	calls int
}

func (f *fakeScanner) Scan(_ context.Context, _ io.Reader) (bool, error) {
	f.calls++
	return f.clean, f.err
}

// TestHandleUploadScannerClean 验证扫描干净时放行落库
func TestHandleUploadScannerClean(t *testing.T) {
	st := newFakeStorage()
	sc := &fakeScanner{clean: true}
	h := NewUploadHandler(UploadConfig{Storage: st, Scanner: sc})
	r := uploadEngine(h)
	r.ServeHTTP(httptest.NewRecorder(),
		multipartFileRequestWithType(t, "file", "a.txt", "text/plain", "hello"))
	assert.Len(t, st.uploaded, 1, "干净文件应落库")
	assert.Equal(t, 1, sc.calls)
}

// TestHandleUploadScannerDirty 验证检测到威胁时拒绝落库（fail-closed）
func TestHandleUploadScannerDirty(t *testing.T) {
	st := newFakeStorage()
	sc := &fakeScanner{clean: false}
	h := NewUploadHandler(UploadConfig{Storage: st, Scanner: sc})
	w := httptest.NewRecorder()
	uploadEngine(h).ServeHTTP(w,
		multipartFileRequestWithType(t, "file", "mal.exe", "application/octet-stream", "X"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, st.uploaded, "感染文件不应落库")
	// CodeInvalidParam=1001（Fail 按错误码返回 i18n 消息，非原始 message）
	assert.Contains(t, w.Body.String(), `"code":1001`)
}

// TestHandleUploadScannerUnavailable 验证扫描不可达时拒绝落库（fail-closed）
func TestHandleUploadScannerUnavailable(t *testing.T) {
	st := newFakeStorage()
	sc := &fakeScanner{err: errors.New("clamd down")}
	h := NewUploadHandler(UploadConfig{Storage: st, Scanner: sc})
	w := httptest.NewRecorder()
	uploadEngine(h).ServeHTTP(w,
		multipartFileRequestWithType(t, "file", "a.txt", "text/plain", "hello"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, st.uploaded, "扫描不可达时不应落库")
}

// startFakeClamd 启动一个 TCP 监听器模拟 clamd：接受连接，读 zINSTREAM 命令与分块帧，回写预设响应。
// 返回监听地址与关闭函数。比 net.Pipe 更贴近真实路径，避开 pipe deadline 怪异。
func startFakeClamd(t *testing.T, response string) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.ReadFull(conn, make([]byte, 10)) // zINSTREAM\0 (10 字节)
		for {
			var sz uint32
			if err := binary.Read(conn, binary.BigEndian, &sz); err != nil {
				return
			}
			if sz == 0 {
				break
			}
			_, _ = io.ReadFull(conn, make([]byte, sz))
		}
		_, _ = conn.Write([]byte(response))
	}()
	return ln.Addr().String(), func() { _ = ln.Close(); <-done }
}

// TestClamAVScannerClean 验证 INSTREAM 协议往返，干净响应
func TestClamAVScannerClean(t *testing.T) {
	addr, cleanup := startFakeClamd(t, "stream: OK\n")
	defer cleanup()

	s := &ClamAVScanner{address: addr, timeout: 5 * time.Second, chunkSize: 4}
	clean, err := s.Scan(context.Background(), strings.NewReader("hello"))
	assert.NoError(t, err)
	assert.True(t, clean)
}

// TestClamAVScannerFound 验证检测到威胁返回 (false, nil)
func TestClamAVScannerFound(t *testing.T) {
	addr, cleanup := startFakeClamd(t, "stream: FOUND EICAR-Test\n")
	defer cleanup()

	s := &ClamAVScanner{address: addr, timeout: 5 * time.Second, chunkSize: 16}
	clean, err := s.Scan(context.Background(), strings.NewReader("bad"))
	assert.NoError(t, err)
	assert.False(t, clean)
}

// TestClamAVScannerDialError 验证连接失败时 fail-closed（返回 false+err）
func TestClamAVScannerDialError(t *testing.T) {
	s := &ClamAVScanner{address: "127.0.0.1:1", timeout: time.Second, chunkSize: 16}
	clean, err := s.Scan(context.Background(), strings.NewReader("x"))
	assert.Error(t, err)
	assert.False(t, clean)
}
