package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func (c *Config) Validate(env string) error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if env != "prod" {
		return nil
	}
	if len(c.Auth.JWTSecret) < 32 || c.Auth.JWTSecret == "change-me-in-production" || strings.Contains(c.Auth.JWTSecret, "${") {
		return errors.New("invalid auth.jwt_secret")
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
	if c.Retention.Enabled {
		if strings.TrimSpace(c.Retention.Cron) == "" {
			return errors.New("invalid retention.cron: required when retention.enabled is true")
		}
		if c.Retention.BatchSize < 0 {
			return errors.New("invalid retention.batch_size: must not be negative")
		}
	}
	if !contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid log.level: %q, must be one of %v", c.Log.Level, validLogLevels)
	}
	if !contains(validLogFormats, c.Log.Format) {
		return fmt.Errorf("invalid log.format: %q, must be one of %v", c.Log.Format, validLogFormats)
	}
	if !contains(validQueueTypes, c.Queue.Type) {
		return fmt.Errorf("invalid queue.type: %q, must be one of %v", c.Queue.Type, validQueueTypes)
	}
	if !contains(validOutboxPublishers, c.Outbox.Publisher) {
		return fmt.Errorf("invalid outbox.publisher: %q, must be one of %v", c.Outbox.Publisher, validOutboxPublishers)
	}
	if !contains(validSchedulerStores, c.Scheduler.Store) {
		return fmt.Errorf("invalid scheduler.store: %q, must be one of %v", c.Scheduler.Store, validSchedulerStores)
	}
	if c.Outbox.Publisher == OutboxPublisherMQ && !contains(validOutboxMQQueueTypes, c.Queue.Type) {
		return fmt.Errorf("invalid queue.type %q for outbox.publisher %q, must be one of %v", c.Queue.Type, c.Outbox.Publisher, validOutboxMQQueueTypes)
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
	if c.Auth.Issuer == "" || c.Auth.AccessExpireMin <= 0 || c.Auth.RefreshExpireDay <= 0 {
		return errors.New("invalid auth configuration")
	}
	if c.Auth.ResetCodeTTLMin <= 0 {
		return errors.New("invalid auth.reset_code_ttl_min")
	}
	if c.Auth.LoginRateLimit <= 0 || c.Auth.LoginRateWindowSec <= 0 || c.Auth.RegisterRateLimit <= 0 || c.Auth.RegisterRateWindowSec <= 0 {
		return errors.New("invalid auth rate limit")
	}
	if c.Auth.Provisioning.Enabled && !c.Auth.PublicRegistration {
		return errors.New("auth.provisioning.enabled requires auth.public_registration")
	}
	if err := validateProvisioning(c.Auth.Provisioning); err != nil {
		return err
	}
	if err := validateOAuthProviders(c.OAuth); err != nil {
		return err
	}
	if err := validateWebAuthn(c.Auth.WebAuthn); err != nil {
		return err
	}
	if c.Security.IdempotencyEnabled && c.Security.IdempotencyTTLSec <= 0 {
		return errors.New("security.idempotency_ttl_sec must be positive when idempotency is enabled")
	}
	if c.Audit.QueueSize <= 0 || c.Audit.BatchSize <= 0 || c.Audit.BatchSize > c.Audit.QueueSize || c.Audit.FlushIntervalMS <= 0 {
		return errors.New("invalid audit configuration")
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
	if c.Captcha.Enabled && c.Captcha.TTLMin <= 0 {
		return errors.New("invalid captcha.ttl_min")
	}
	return nil
}

// validateProvisioning 校验开通式注册配置：enabled 时要求公开注册开启、模板非空、
// 角色名唯一且权限条目完整、owner_role 必须能在模板中解析（空 = 第一个角色）
func validateProvisioning(p ProvisioningConfig) error {
	if !p.Enabled {
		return nil
	}
	if len(p.Roles) == 0 {
		return errors.New("auth.provisioning.enabled requires at least one role in auth.provisioning.roles")
	}
	names := make(map[string]bool, len(p.Roles))
	for _, role := range p.Roles {
		if role.Name == "" {
			return errors.New("auth.provisioning.roles[].name is required")
		}
		if names[role.Name] {
			return fmt.Errorf("duplicate auth.provisioning.roles[].name: %q", role.Name)
		}
		names[role.Name] = true
		for _, perm := range role.Permissions {
			if perm.Resource == "" || perm.Action == "" {
				return fmt.Errorf("auth.provisioning.roles[%q].permissions entries require resource and action", role.Name)
			}
		}
	}
	if p.OwnerRole != "" && !names[p.OwnerRole] {
		return fmt.Errorf("auth.provisioning.owner_role %q not found in auth.provisioning.roles", p.OwnerRole)
	}
	return nil
}

// validateOAuthProviders 校验启用的 OAuth/OIDC 提供商：client_id/redirect_url 必填，
// OIDC（配了 issuer_url）还要求 issuer_url 是 http(s) 绝对地址。
func validateOAuthProviders(cfg OAuthConfig) error {
	for name, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		if p.ClientID == "" || p.RedirectURL == "" {
			return fmt.Errorf("oauth.providers.%s requires client_id and redirect_url when enabled", name)
		}
		if p.IssuerURL == "" {
			continue
		}
		u, err := url.Parse(p.IssuerURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("oauth.providers.%s.issuer_url must be an absolute http(s) URL", name)
		}
	}
	return nil
}

// validateWebAuthn 校验启用的 WebAuthn 配置：rp_id 必填，rp_origins 必须是非空绝对 http(s) 来源
func validateWebAuthn(cfg WebAuthnConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.RPID) == "" {
		return errors.New("auth.webauthn.rp_id is required when enabled")
	}
	if len(cfg.RPOrigins) == 0 {
		return errors.New("auth.webauthn.rp_origins is required when enabled")
	}
	for _, origin := range cfg.RPOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("auth.webauthn.rp_origins entry %q must be an absolute http(s) origin", origin)
		}
	}
	if cfg.SessionTTLMin < 0 {
		return errors.New("auth.webauthn.session_ttl_min must not be negative")
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
