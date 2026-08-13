package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/observability"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatAddr(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"full", "0.0.0.0", 8080, "0.0.0.0:8080"},
		{"zero port", "127.0.0.1", 0, "127.0.0.1:0"},
		{"empty host", "", 9090, ":9090"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatAddr(tt.host, tt.port))
		})
	}
}

func TestNewServerPlain(t *testing.T) {
	cfg := config.HTTPConfig{
		Host: "127.0.0.1", Port: 8080,
		ReadHeaderTimeoutSec: 5, ReadTimeoutSec: 10, WriteTimeoutSec: 15, IdleTimeoutSec: 30,
	}
	engine := gin.New()
	srv, err := NewServer(cfg, engine)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8080", srv.Addr)
	assert.Equal(t, engine, srv.Handler)
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, srv.ReadTimeout)
	assert.Equal(t, 15*time.Second, srv.WriteTimeout)
	assert.Equal(t, 30*time.Second, srv.IdleTimeout)
	assert.Nil(t, srv.TLSConfig)
}

func TestNewServerTLSInvalidFiles(t *testing.T) {
	cfg := config.HTTPConfig{
		Host: "127.0.0.1", Port: 8443,
		TLS: config.TLSConfig{Enabled: true, CertFile: "/nonexistent/cert.pem", KeyFile: "/nonexistent/key.pem"},
	}
	_, err := NewServer(cfg, gin.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load tls key pair")
}

func TestNewServerTLSValid(t *testing.T) {
	certFile, keyFile := writeTestCert(t)
	cfg := config.HTTPConfig{
		Host: "127.0.0.1", Port: 8443,
		TLS: config.TLSConfig{Enabled: true, CertFile: certFile, KeyFile: keyFile},
	}
	srv, err := NewServer(cfg, gin.New())
	require.NoError(t, err)
	require.NotNil(t, srv.TLSConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), srv.TLSConfig.MinVersion)
	assert.Len(t, srv.TLSConfig.Certificates, 1)
}

func writeTestCert(t *testing.T) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyFile := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
	return certFile, keyFile
}

func testLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "error", Format: "console", Output: "stdout"})
}

func TestSetupRouterWiresGlobalMiddleware(t *testing.T) {
	r := SetupRouter(testLogger(),
		config.HTTPConfig{Mode: config.HTTPModeTest, AllowedOrigins: []string{"*"}},
		config.ServerConfig{TimeoutSec: 30, RateLimitRate: 100, RateLimitBurst: 200},
		config.SecurityConfig{},
		observability.TracingConfig{Enabled: false},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(stdhttp.MethodGet, "/nope", nil))
	assert.Equal(t, stdhttp.StatusNotFound, w.Code)
	// 全局中间件生效：请求 ID、安全头
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age")
}

func TestSetupRouterCSRFDisabledByDefault(t *testing.T) {
	r := SetupRouter(testLogger(),
		config.HTTPConfig{Mode: config.HTTPModeTest},
		config.ServerConfig{},
		config.SecurityConfig{},
		observability.TracingConfig{},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(stdhttp.MethodPost, "/nope", nil))
	assert.Equal(t, stdhttp.StatusNotFound, w.Code)
}

func TestSetupRouterEnablesCSRFWhenSecretSet(t *testing.T) {
	secCfg := config.SecurityConfig{CSRFSecret: "csrf-secret"}
	r := SetupRouter(testLogger(),
		config.HTTPConfig{Mode: config.HTTPModeTest},
		config.ServerConfig{},
		secCfg,
		observability.TracingConfig{},
	)

	// 无 cookie 的 POST 被 CSRF 拦截
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(stdhttp.MethodPost, "/nope", nil))
	assert.Equal(t, stdhttp.StatusForbidden, w.Code)

	// Bearer 请求跳过 CSRF
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(stdhttp.MethodPost, "/nope", nil)
	req2.Header.Set("Authorization", "Bearer token.xyz")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, stdhttp.StatusNotFound, w2.Code)
}

func TestSetupRouterOTELEnabled(t *testing.T) {
	r := SetupRouter(testLogger(),
		config.HTTPConfig{Mode: config.HTTPModeTest},
		config.ServerConfig{},
		config.SecurityConfig{},
		observability.TracingConfig{Enabled: true, ServiceName: "test-svc"},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(stdhttp.MethodGet, "/x", nil))
	assert.Equal(t, stdhttp.StatusNotFound, w.Code)
}

func TestConfigureTrustedProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	require.NoError(t, ConfigureTrustedProxies(r, []string{"127.0.0.1", "10.0.0.0/8"}))
	require.NoError(t, ConfigureTrustedProxies(r, nil))
	require.NoError(t, ConfigureTrustedProxies(r, []string{}))
	require.Error(t, ConfigureTrustedProxies(r, []string{"127.0.0.1/999"}))
}

func TestServerStartStopServesRequests(t *testing.T) {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/ping", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	addr := freeAddr(t)
	srv := newServer(&stdhttp.Server{Addr: addr, Handler: mux})

	require.NoError(t, srv.Start(context.Background()))

	var body []byte
	require.Eventually(t, func() bool {
		resp, err := stdhttp.Get("http://" + addr + "/ping")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != stdhttp.StatusOK {
			return false
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}
		body = b
		return true
	}, 3*time.Second, 20*time.Millisecond)
	assert.Equal(t, "pong", string(body))

	require.NoError(t, srv.Stop(context.Background()))
}

func TestServerStartReturnsBindError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	srv := newServer(&stdhttp.Server{Addr: ln.Addr().String(), Handler: stdhttp.NewServeMux()})
	err = srv.Start(context.Background())
	require.Error(t, err)
}

func TestNewManagementServer(t *testing.T) {
	cfg := config.ManagementConfig{Host: "127.0.0.1", Port: 9090}
	mux := stdhttp.NewServeMux()
	srv := NewManagementServer(cfg, mux)
	assert.Equal(t, "127.0.0.1:9090", srv.Addr)
	assert.Equal(t, mux, srv.Handler)
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, srv.ReadTimeout)
	assert.Equal(t, 10*time.Second, srv.WriteTimeout)
	assert.Equal(t, 30*time.Second, srv.IdleTimeout)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}
