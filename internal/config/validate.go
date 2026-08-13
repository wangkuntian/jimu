package config

import (
	"errors"
	"fmt"
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
	if c.Captcha.Enabled && c.Captcha.TTLMin <= 0 {
		return errors.New("invalid captcha.ttl_min")
	}
	return nil
}
