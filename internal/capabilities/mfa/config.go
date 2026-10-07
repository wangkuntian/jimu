package mfa

import (
	"errors"
	"strings"

	"jimu/internal/config"
)

const ConfigKey = "mfa"

// Settings is the YAML configuration owned by the MFA capability.
type Settings struct {
	TrustedDeviceDays int             `mapstructure:"trusted_device_days"`
	Retention         RetentionConfig `mapstructure:"retention"`
}

type RetentionConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	Cron              string `mapstructure:"cron"`
	BatchSize         int    `mapstructure:"batch_size"`
	ExpiredDeviceDays int    `mapstructure:"expired_device_days"`
}

func (s *Settings) ApplyDefaults() {}

func (s Settings) Validate() error {
	if s.TrustedDeviceDays < 0 {
		return errors.New("mfa.trusted_device_days must not be negative")
	}
	if s.Retention.Enabled && strings.TrimSpace(s.Retention.Cron) == "" {
		return errors.New("mfa.retention.cron is required when enabled")
	}
	if s.Retention.BatchSize < 0 || s.Retention.ExpiredDeviceDays < 0 {
		return errors.New("mfa.retention values must not be negative")
	}
	return nil
}

var _ config.SectionConfig = (*Settings)(nil)

// Config mfa 能力的装配期配置（不来自 YAML 段）。
//
// mfa 的 Requires 只有 user（不依赖 auth），故**不得** import auth 能力类型；
// 本结构由组合根从 auth 段取值后传入（JWT 参数用于保护本能力的路由，
// TrustedDeviceDays/Issuer 用于可信设备与 TOTP 开户 URI）。
type Config struct {
	JWTSecret         string
	JWTPreviousSecret string
	Issuer            string
	AccessExpireMin   int
	RefreshExpireDay  int
	TrustedDeviceDays int
}
