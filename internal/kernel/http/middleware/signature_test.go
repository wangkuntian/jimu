package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var testSecret = []byte("test-signing-secret")

func signatureRouter(cfg SignatureConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Signature(cfg))
	r.POST("/api/v1/upload", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func TestDefaultSignatureConfig(t *testing.T) {
	cfg := DefaultSignatureConfig(testSecret)
	assert.Equal(t, "X-Api-Key", cfg.HeaderKey)
	assert.Equal(t, "X-Signature", cfg.HeaderSign)
	assert.Equal(t, "X-Timestamp", cfg.HeaderTimestamp)
	assert.Equal(t, "X-Nonce", cfg.HeaderNonce)
	assert.Equal(t, 5*time.Minute, cfg.MaxAge)
}

func TestSignatureMissingHeaders(t *testing.T) {
	r := signatureRouter(DefaultSignatureConfig(testSecret))
	base := map[string]string{
		"X-Api-Key": "key", "X-Signature": "sig", "X-Timestamp": "1", "X-Nonce": "nonce",
	}
	for _, omit := range []string{"X-Api-Key", "X-Signature", "X-Timestamp", "X-Nonce"} {
		t.Run("missing "+omit, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
			for k, v := range base {
				if k != omit {
					req.Header.Set(k, v)
				}
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestSignatureInvalidTimestamp(t *testing.T) {
	r := signatureRouter(DefaultSignatureConfig(testSecret))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
	req.Header.Set("X-Api-Key", "k")
	req.Header.Set("X-Signature", "sig")
	req.Header.Set("X-Timestamp", "not-a-number")
	req.Header.Set("X-Nonce", "n")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSignatureExpired(t *testing.T) {
	r := signatureRouter(DefaultSignatureConfig(testSecret))
	for _, ts := range []int64{time.Now().Unix() - 600, time.Now().Unix() + 600} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
		req.Header.Set("X-Api-Key", "k")
		req.Header.Set("X-Signature", "sig")
		req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
		req.Header.Set("X-Nonce", "n")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "timestamp=%d", ts)
	}
}

func TestSignatureValidWithQueryAndBody(t *testing.T) {
	r := signatureRouter(DefaultSignatureConfig(testSecret))

	timestamp := time.Now().Unix()
	nonce := "nonce-123"
	body := []byte(`{"name":"hello"}`)
	query := map[string][]string{"b": {"2"}, "a": {"1"}}

	sig := SignRequest(testSecret, "POST", "/api/v1/upload", query, body, timestamp, nonce)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload?a=1&b=2", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "key")
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSignatureInvalidSignature(t *testing.T) {
	r := signatureRouter(DefaultSignatureConfig(testSecret))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
	req.Header.Set("X-Api-Key", "key")
	req.Header.Set("X-Signature", "wrong-signature")
	req.Header.Set("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("X-Nonce", "nonce")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSignatureSkipper(t *testing.T) {
	cfg := DefaultSignatureConfig(testSecret)
	cfg.Skipper = func(c *gin.Context) bool { return true }
	r := signatureRouter(cfg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestBuildSignString(t *testing.T) {
	got := buildSignString("post", "/path",
		map[string][]string{"z": {"1"}, "a": {"2", "1"}},
		[]byte("body"), "1000", "n")
	// METHOD(大写) + PATH + 排序后的 query(k+v 扁平) + body + timestamp + nonce
	assert.Equal(t, "POST/patha1a2z1body1000n", got)
}

func TestHmacSign(t *testing.T) {
	s1 := hmacSign(testSecret, "data")
	s2 := hmacSign(testSecret, "data")
	assert.Len(t, s1, 64)
	assert.Equal(t, s1, s2)
}
