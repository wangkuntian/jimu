package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

func (c *Config) Validate(env string) error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if env != "prod" {
		return nil
	}
	if c.DB.Password == "" || c.DB.Password == "root" || strings.Contains(c.DB.Password, "${") {
		return errors.New("invalid db.password")
	}
	if c.Management.Port < 1 || c.Management.Port > 65535 {
		return errors.New("invalid management.port")
	}
	for _, origin := range c.HTTP.AllowedOrigins {
		if origin == "*" {
			return errors.New("invalid http.allowed_origins")
		}
	}
	if c.Security.EncryptionKey != "" && len(c.Security.EncryptionKey) < 32 {
		return errors.New("invalid security.encryption_key, must be at least 32 bytes when set")
	}
	return nil
}

func (c *Config) validateCommon() error {
	if !contains(validHTTPModes, c.HTTP.Mode) {
		return fmt.Errorf("invalid http.mode: %q, must be one of %v", c.HTTP.Mode, validHTTPModes)
	}
	if !contains(validDBDrivers, c.DB.Driver) {
		return fmt.Errorf("invalid db.driver: %q, must be one of %v", c.DB.Driver, validDBDrivers)
	}
	if err := validateTLS("http.tls", c.HTTP.TLS); err != nil {
		return err
	}
	if err := validateTLS("grpc.tls", c.GRPC.TLS); err != nil {
		return err
	}
	if err := validateCIDRs("security.ip_allowlist", c.Security.IPAllowlist); err != nil {
		return err
	}
	if err := validateCIDRs("security.admin_ip_allowlist", c.Security.AdminIPAllowlist); err != nil {
		return err
	}
	if !contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid log.level: %q, must be one of %v", c.Log.Level, validLogLevels)
	}
	if !contains(validLogFormats, c.Log.Format) {
		return fmt.Errorf("invalid log.format: %q, must be one of %v", c.Log.Format, validLogFormats)
	}
	if c.ID.WorkerID < 0 || c.ID.WorkerID > 1023 {
		return errors.New("invalid id.worker_id, must be 0-1023")
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		return errors.New("invalid http.port")
	}
	if c.HTTP.ReadHeaderTimeoutSec <= 0 || c.HTTP.ReadTimeoutSec <= 0 || c.HTTP.WriteTimeoutSec <= 0 || c.HTTP.IdleTimeoutSec <= 0 || c.HTTP.ShutdownTimeoutSec <= 0 {
		return errors.New("invalid http timeout")
	}
	if c.HTTP.MaxBodyBytes <= 0 {
		return errors.New("invalid http.max_body_bytes")
	}
	if c.Management.ProbeTimeoutSec <= 0 {
		return errors.New("invalid management.probe_timeout_sec")
	}
	if c.Security.IdempotencyEnabled && c.Security.IdempotencyTTLSec <= 0 {
		return errors.New("security.idempotency_ttl_sec must be positive when idempotency is enabled")
	}
	if c.DB.MaxOpen <= 0 || c.DB.MaxIdle <= 0 || c.DB.MaxIdle > c.DB.MaxOpen {
		return errors.New("invalid db pool configuration")
	}
	if c.DB.MaxRetries <= 0 || c.DB.RetryIntervalSec <= 0 {
		return errors.New("invalid db retry configuration")
	}
	if c.Redis.MaxRetries <= 0 || c.Redis.RetryIntervalSec <= 0 {
		return errors.New("invalid redis retry configuration")
	}
	if c.Redis.Mode == "" {
		c.Redis.Mode = RedisModeSingle
	}
	if !contains(validRedisModes, c.Redis.Mode) {
		return fmt.Errorf("invalid redis.mode: %q, must be one of %v", c.Redis.Mode, validRedisModes)
	}
	switch c.Redis.Mode {
	case RedisModeSentinel:
		if c.Redis.MasterName == "" || len(c.Redis.SentinelAddrs) == 0 {
			return errors.New("invalid redis sentinel config: master_name and sentinel_addrs are required")
		}
	case RedisModeCluster:
		if len(c.Redis.ClusterAddrs) == 0 {
			return errors.New("invalid redis cluster config: cluster_addrs is required")
		}
	}
	if err := validateCapabilities(&c.Capabilities); err != nil {
		return err
	}
	return nil
}

// validateCIDRs 校验 IP 白名单条目（CIDR 或单个 IP），非法值启动即报错
func validateCIDRs(key string, entries []string) error {
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if net.ParseIP(entry) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err != nil {
			return fmt.Errorf("invalid %s entry %q: must be an IP or CIDR", key, entry)
		}
	}
	return nil
}

// validateTLS 校验 TLS 配置一致性：启用时必须提供证书与私钥
func validateTLS(key string, cfg TLSConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return fmt.Errorf("invalid %s: cert_file and key_file are required when enabled", key)
	}
	return nil
}

// validateCapabilities 校验能力开关：名称非空且不重复（能力名是否存在由 catalog 解析时校验）。
// 归一化后的名字写回配置，避免 " user" 通过校验后在 catalog.Resolve 处报 unknown capability。
func validateCapabilities(cfg *CapabilitiesConfig) error {
	seen := make(map[string]bool, len(cfg.Enabled))
	for i, raw := range cfg.Enabled {
		name := strings.TrimSpace(raw)
		if name == "" {
			return fmt.Errorf("invalid capabilities.enabled[%d]: name must not be blank", i)
		}
		if seen[name] {
			return fmt.Errorf("duplicate capabilities.enabled entry: %q", name)
		}
		seen[name] = true
		cfg.Enabled[i] = name
	}
	return nil
}
