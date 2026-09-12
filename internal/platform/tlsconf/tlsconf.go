// Package tlsconf 依据配置构建 tls.Config，供 HTTP 与 gRPC server 共用：
// 支持服务端证书，以及可选的双向认证（mTLS：要求并校验客户端证书）。
package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"jimu/internal/config"
)

// ServerConfig 构建服务端 tls.Config：
//   - 未启用时返回 (nil, nil)
//   - 必须提供 cert_file 与 key_file
//   - client_ca_file 非空时启用 mTLS：要求客户端出示证书并用该 CA 校验
func ServerConfig(cfg config.TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, errors.New("tls: cert_file and key_file are required when tls is enabled")
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: load key pair: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if cfg.ClientCAFile != "" {
		pool, err := LoadCAPool(cfg.ClientCAFile)
		if err != nil {
			return nil, err
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsCfg, nil
}

// LoadCAPool 读取 CA 证书文件并构建证书池（用于校验客户端或服务端证书）
func LoadCAPool(path string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tls: read ca file %s: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("tls: no valid certificate found in %s", path)
	}
	return pool, nil
}
