package authmodule

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键（原 config.AuthConfig，P2.1 下沉）。
//
// 设计 §8 ¶2：`auth.webauthn.enabled` / `auth.provisioning.enabled` 属「保留为能力内配置」
// 的开关，只有能力级开关改由 capabilities.enabled 表达，因此本段**不拆**：
// auth 拥有整个 auth 段（含嵌套 webauthn/provisioning），对应 §6.1「一个能力一份 Config」。
const ConfigKey = "auth"

// Config 认证能力配置段。
type Config struct {
	JWTSecret             string             `mapstructure:"jwt_secret"`
	JWTPreviousSecret     string             `mapstructure:"jwt_previous_secret"`
	Issuer                string             `mapstructure:"issuer"`
	AccessExpireMin       int                `mapstructure:"access_expire_min"`
	RefreshExpireDay      int                `mapstructure:"refresh_expire_day"`
	PublicRegistration    bool               `mapstructure:"public_registration"`
	LoginRateLimit        int                `mapstructure:"login_rate_limit"`
	LoginRateWindowSec    int                `mapstructure:"login_rate_window_sec"`
	RegisterRateLimit     int                `mapstructure:"register_rate_limit"`
	RegisterRateWindowSec int                `mapstructure:"register_rate_window_sec"`
	ResetCodeTTLMin       int                `mapstructure:"reset_code_ttl_min"`     // 密码重置验证码有效期（分钟）
	PasswordHistoryCount  int                `mapstructure:"password_history_count"` // 防复用：检查最近 N 个历史密码（0=关闭）
	TrustedDeviceDays     int                `mapstructure:"trusted_device_days"`    // 可信设备有效期（天，0=关闭「记住此设备」）
	BreachCheckEnabled    bool               `mapstructure:"breach_check_enabled"`   // 泄露口令检查（HIBP k-匿名范围查询，默认关闭）
	Provisioning          ProvisioningConfig `mapstructure:"provisioning"`           // 开通式注册（注册 = 开通新租户）
	WebAuthn              WebAuthnConfig     `mapstructure:"webauthn"`               // WebAuthn/通行密钥（无密码登录）
}

// WebAuthnConfig WebAuthn/通行密钥配置。
// rp_id 必须是站点有效域（不带 scheme，如 example.com；本地开发用 localhost），
// rp_origins 是允许的浏览器来源（含 scheme，如 https://example.com）。
type WebAuthnConfig struct {
	Enabled       bool     `mapstructure:"enabled"`         // 是否启用通行密钥
	RPDisplayName string   `mapstructure:"rp_display_name"` // 展示给用户的站点名称
	RPID          string   `mapstructure:"rp_id"`           // Relying Party ID（站点有效域）
	RPOrigins     []string `mapstructure:"rp_origins"`      // 允许的来源（绝对 URL）
	SessionTTLMin int      `mapstructure:"session_ttl_min"` // 挑战有效期（分钟），0 用默认 5
}

// ProvisioningConfig 开通式注册配置。
// enabled 时 /auth/register 在单事务内创建新租户 + owner 用户，并按 roles 模板
// 初始化租户角色与全局权限绑定；owner 获得绑定 owner_role 指定的角色（缺省为模板第一个角色）。
type ProvisioningConfig struct {
	Enabled   bool                    `mapstructure:"enabled"`
	OwnerRole string                  `mapstructure:"owner_role"` // owner 绑定的模板角色名；空 = 模板第一个角色
	Roles     []ProvisionRoleTemplate `mapstructure:"roles"`
}

// ProvisionRoleTemplate 开通租户时初始化的角色模板。
// permissions 引用全局权限表（seed 写入的 resource + action），缺失的权限跳过不报错。
type ProvisionRoleTemplate struct {
	Name        string                `mapstructure:"name"`
	Description string                `mapstructure:"description"`
	Permissions []ProvisionPermission `mapstructure:"permissions"`
}

// ProvisionPermission 模板角色绑定的全局权限
type ProvisionPermission struct {
	Resource string `mapstructure:"resource"`
	Action   string `mapstructure:"action"`
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

// Validate 校验本能力配置段（原 config.validateCommon 的 auth 检查 + validateProvisioning
// + validateWebAuthn）。仅在本能力**启用**时由组合根调用，因此未启用能力的配置段
// 既不出现也不校验（设计 §8）。
//
// 注意：`provisioning.enabled` 要求 `public_registration` 的跨字段校验刻意不在此处
// （P2.1 裁定：随 provisioning 未来归属变化，留在组合根，见 cmd/server/main.go）。
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
	if err := validateProvisioning(c.Provisioning); err != nil {
		return err
	}
	if err := validateWebAuthn(c.WebAuthn); err != nil {
		return err
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

// validateProvisioning 校验开通式注册配置：enabled 时要求模板非空、
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
