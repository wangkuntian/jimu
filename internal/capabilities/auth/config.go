package authmodule

import (
	"errors"
	"strings"

	"jimu/internal/config"
	"jimu/internal/contract"
)

func (c Config) PortView() contract.AuthConfig {
	return contract.AuthConfig{JWTSecret: c.JWTSecret, JWTPreviousSecret: c.JWTPreviousSecret, Issuer: c.Issuer,
		AccessExpireMin: c.AccessExpireMin, RefreshExpireDay: c.RefreshExpireDay, PublicRegistration: c.PublicRegistration,
		LoginRateLimit: c.LoginRateLimit, LoginRateWindowSec: c.LoginRateWindowSec,
		RegisterRateLimit: c.RegisterRateLimit, RegisterRateWindowSec: c.RegisterRateWindowSec,
		BreachCheckEnabled: c.BreachCheckEnabled}
}

// ConfigKey 本能力在 app.yaml 中的配置段键（原 config.AuthConfig，P2.1 下沉）。
const ConfigKey = "auth"

// Config 认证能力配置段。
type Config struct {
	JWTSecret             string `mapstructure:"jwt_secret"`
	JWTPreviousSecret     string `mapstructure:"jwt_previous_secret"`
	Issuer                string `mapstructure:"issuer"`
	AccessExpireMin       int    `mapstructure:"access_expire_min"`
	RefreshExpireDay      int    `mapstructure:"refresh_expire_day"`
	PublicRegistration    bool   `mapstructure:"public_registration"`
	LoginRateLimit        int    `mapstructure:"login_rate_limit"`
	LoginRateWindowSec    int    `mapstructure:"login_rate_window_sec"`
	RegisterRateLimit     int    `mapstructure:"register_rate_limit"`
	RegisterRateWindowSec int    `mapstructure:"register_rate_window_sec"`
	ResetCodeTTLMin       int    `mapstructure:"reset_code_ttl_min"`     // 密码重置验证码有效期（分钟）
	PasswordHistoryCount  int    `mapstructure:"password_history_count"` // 防复用：检查最近 N 个历史密码（0=关闭）
	BreachCheckEnabled    bool   `mapstructure:"breach_check_enabled"`   // 泄露口令检查（HIBP k-匿名范围查询，默认关闭）
}

// ApplyDefaults 应用本能力段的环境变量覆盖（原 config.applyEnvOverrides 的 auth 两项）：
// JWT_SECRET(_FILE) / JWT_PREVIOUS_SECRET(_FILE) 优先于 YAML，兼容 Docker Secrets。
func (c *Config) ApplyDefaults() {
	if v := config.GetEnvOrFile("JWT_SECRET_FILE", "JWT_SECRET"); v != "" {
		c.JWTSecret = v
	}
	if v := config.GetEnvOrFile("JWT_PREVIOUS_SECRET_FILE", "JWT_PREVIOUS_SECRET"); v != "" {
		c.JWTPreviousSecret = v
	}
}

// Validate 校验本能力配置段。仅在本能力**启用**时由组合根调用，因此未启用能力的配置段
// 既不出现也不校验（设计 §8）。
func (c *Config) Validate() error {
	if c.Issuer == "" || c.AccessExpireMin <= 0 || c.RefreshExpireDay <= 0 {
		return errors.New("invalid auth configuration")
	}
	if c.ResetCodeTTLMin <= 0 {
		return errors.New("invalid auth.reset_code_ttl_min")
	}
	if c.LoginRateLimit <= 0 || c.LoginRateWindowSec <= 0 || c.RegisterRateLimit <= 0 || c.RegisterRateWindowSec <= 0 {
		return errors.New("invalid auth rate limit")
	}
	return nil
}

// ValidateProd 生产环境加严校验（config.ProdConfigValidator，由 app.LoadCapabilityConfigs
// 在 APP_ENV=prod 时按类型断言调用）：jwt_secret 必须足够强且非占位/未展开值。
func (c *Config) ValidateProd() error {
	if len(c.JWTSecret) < 32 || c.JWTSecret == "change-me-in-production" || strings.Contains(c.JWTSecret, "${") {
		return errors.New("invalid auth.jwt_secret")
	}
	return nil
}
