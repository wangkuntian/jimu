package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	if c.DB.Timezone != "" {
		if _, err := time.LoadLocation(c.DB.Timezone); err != nil {
			return fmt.Errorf("invalid db.timezone: %q is not a valid IANA time zone", c.DB.Timezone)
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
