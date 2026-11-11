package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GzipCompression())
	r.GET("/data", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain", bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 100))
	})
	return r
}

func TestGzipCompressionNoAcceptEncoding(t *testing.T) {
	r := gzipRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/data", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
}

func TestGzipCompressionEncodesBody(t *testing.T) {
	r := gzipRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))

	zr, err := gzip.NewReader(w.Body)
	require.NoError(t, err)
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Len(t, decoded, 26*100)
}

func TestGzipCompressionSkipsAlreadyCompressed(t *testing.T) {
	r := gzipRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "image/png")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Len(t, w.Body.Bytes(), 26*100)
}

func TestIsAlreadyCompressed(t *testing.T) {
	for _, ct := range []string{
		"image/png", "audio/mp3", "video/mp4", "application/gzip",
		"application/zip", "application/x-7z-compressed", "application/x-rar-compressed", "application/pdf",
	} {
		assert.True(t, isAlreadyCompressed(ct), "content-type=%s", ct)
	}
	for _, ct := range []string{"text/plain", "application/json", ""} {
		assert.False(t, isAlreadyCompressed(ct), "content-type=%s", ct)
	}
}
