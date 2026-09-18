package http

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
)

// noopScanner 零开销扫描器，用于基准对照（隔离扫描本身开销与上传路径开销）。
type noopScanner struct{}

func (noopScanner) Scan(_ context.Context, _ io.Reader) (bool, error) { return true, nil }

// benchUploadRequest 构造指定大小的单文件 multipart 请求（bench 专用，不依赖 *testing.T）。
func benchUploadRequest(size int) *http.Request {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="bench.dat"`)
	h.Set("Content-Type", "application/octet-stream")
	fw, err := mw.CreatePart(h)
	if err != nil {
		panic(err)
	}
	_, _ = fw.Write(bytes.Repeat([]byte("x"), size))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// BenchmarkUploadWithoutScanner 基线：上传无扫描（大小校验 + magic-byte 嗅探 + 落存储）。
func BenchmarkUploadWithoutScanner(b *testing.B) {
	h := NewUploadHandler(UploadConfig{Storage: newFakeStorage()})
	r := uploadEngine(h)
	req := benchUploadRequest(4 * 1024) // 4KB

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
	}
}

// BenchmarkUploadWithScanner 上传带零开销扫描器，量化扫描路径引入的固定开销。
func BenchmarkUploadWithScanner(b *testing.B) {
	h := NewUploadHandler(UploadConfig{Storage: newFakeStorage(), Scanner: noopScanner{}})
	r := uploadEngine(h)
	req := benchUploadRequest(4 * 1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
	}
}

// BenchmarkReadAllAndSniff 基准：readAll + magic-byte 嗅探（上传路径核心开销）。
func BenchmarkReadAllAndSniff(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 4*1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := readAll(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			b.Fatal(err)
		}
		_ = http.DetectContentType(buf)
	}
}
