package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TLSMaterial 测试用 mTLS 证书材料：自签 CA + 服务端证书 + 客户端证书（均为文件路径）
type TLSMaterial struct {
	ServerCertFile string
	ServerKeyFile  string
	ClientCAFile   string
	ClientCertFile string
	ClientKeyFile  string
}

// NewTLSMaterial 生成测试用证书材料，写入 t.TempDir()。
// 服务端证书 SAN 覆盖 localhost/127.0.0.1，客户端证书由同一 CA 签发。
func NewTLSMaterial(t *testing.T) *TLSMaterial {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "jimu-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca cert: %v", err)
	}

	serverCertFile, serverKeyFile := issueCert(t, dir, "server", caCert, caKey, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	clientCertFile, clientKeyFile := issueCert(t, dir, "client", caCert, caKey, nil, nil)

	clientCAFile := filepath.Join(dir, "ca.pem")
	writePEM(t, clientCAFile, "CERTIFICATE", caDER)

	return &TLSMaterial{
		ServerCertFile: serverCertFile,
		ServerKeyFile:  serverKeyFile,
		ClientCAFile:   clientCAFile,
		ClientCertFile: clientCertFile,
		ClientKeyFile:  clientKeyFile,
	}
}

// issueCert 用 CA 签发叶子证书，返回证书与私钥文件路径
func issueCert(t *testing.T, dir, name string, ca *x509.Certificate, caKey *ecdsa.PrivateKey, dnsNames []string, ips []net.IP) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate %s key: %v", name, err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "jimu-test-" + name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create %s cert: %v", name, err)
	}

	certFile := filepath.Join(dir, name+".pem")
	keyFile := filepath.Join(dir, name+"-key.pem")
	writePEM(t, certFile, "CERTIFICATE", der)

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal %s key: %v", name, err)
	}
	writePEM(t, keyFile, "EC PRIVATE KEY", keyDER)
	return certFile, keyFile
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	buf := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
