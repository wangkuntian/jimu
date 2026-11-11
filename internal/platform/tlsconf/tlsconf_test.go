package tlsconf

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/config"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfigDisabled(t *testing.T) {
	cfg, err := ServerConfig(config.TLSConfig{})
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestServerConfigRequiresCertAndKey(t *testing.T) {
	_, err := ServerConfig(config.TLSConfig{Enabled: true})
	assert.ErrorContains(t, err, "cert_file and key_file are required")

	_, err = ServerConfig(config.TLSConfig{Enabled: true, CertFile: "cert.pem"})
	assert.ErrorContains(t, err, "cert_file and key_file are required")
}

func TestServerConfigEnablesMTLSWithClientCA(t *testing.T) {
	m := testutil.NewTLSMaterial(t)

	cfg, err := ServerConfig(config.TLSConfig{
		Enabled:      true,
		CertFile:     m.ServerCertFile,
		KeyFile:      m.ServerKeyFile,
		ClientCAFile: m.ClientCAFile,
	})
	require.NoError(t, err)
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth, "配置客户端 CA 应要求并校验客户端证书")
	assert.NotNil(t, cfg.ClientCAs)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	assert.Len(t, cfg.Certificates, 1)
}

func TestServerConfigWithoutClientCAIsServerOnly(t *testing.T) {
	m := testutil.NewTLSMaterial(t)

	cfg, err := ServerConfig(config.TLSConfig{Enabled: true, CertFile: m.ServerCertFile, KeyFile: m.ServerKeyFile})
	require.NoError(t, err)
	assert.Equal(t, tls.NoClientCert, cfg.ClientAuth, "未配置客户端 CA 时不做双向认证")
	assert.Nil(t, cfg.ClientCAs)
}

func TestServerConfigRejectsBadCA(t *testing.T) {
	m := testutil.NewTLSMaterial(t)

	badPEM := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(badPEM, []byte("not a certificate"), 0o600))
	_, err := ServerConfig(config.TLSConfig{Enabled: true, CertFile: m.ServerCertFile, KeyFile: m.ServerKeyFile, ClientCAFile: badPEM})
	assert.ErrorContains(t, err, "no valid certificate")

	_, err = ServerConfig(config.TLSConfig{Enabled: true, CertFile: m.ServerCertFile, KeyFile: m.ServerKeyFile, ClientCAFile: filepath.Join(t.TempDir(), "missing.pem")})
	assert.ErrorContains(t, err, "read ca file")
}

func TestServerConfigRejectsBadKeyPair(t *testing.T) {
	m := testutil.NewTLSMaterial(t)
	_, err := ServerConfig(config.TLSConfig{Enabled: true, CertFile: m.ServerCertFile, KeyFile: m.ClientKeyFile})
	assert.ErrorContains(t, err, "load key pair")
}

// TestMTLSHandshake 用真实握手验证：无客户端证书被拒，带客户端证书放行。
func TestMTLSHandshake(t *testing.T) {
	m := testutil.NewTLSMaterial(t)
	serverTLS, err := ServerConfig(config.TLSConfig{
		Enabled:      true,
		CertFile:     m.ServerCertFile,
		KeyFile:      m.ServerKeyFile,
		ClientCAFile: m.ClientCAFile,
	})
	require.NoError(t, err)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = serverTLS
	srv.StartTLS()
	defer srv.Close()

	caPool, err := LoadCAPool(m.ClientCAFile)
	require.NoError(t, err)

	// 不带客户端证书 → 握手失败
	noCertClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caPool, MinVersion: tls.VersionTLS12}}}
	resp, err := noCertClient.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("request without client certificate should fail the handshake")
	}

	// 带客户端证书 → 200
	clientCert, err := tls.LoadX509KeyPair(m.ClientCertFile, m.ClientKeyFile)
	require.NoError(t, err)
	withCertClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
	}}}
	resp, err = withCertClient.Get(srv.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
