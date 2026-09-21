package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port == 0 {
		t.Error("expected HTTP.Port to be set")
	}
}

func TestJWTSecretOverride(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Auth.JWTSecret != strings.Repeat("a", 32) {
		t.Errorf("expected JWT_SECRET to override, got %q", cfg.Auth.JWTSecret)
	}
}

func TestDBPasswordOverride(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret-from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DB.Password != "secret-from-env" {
		t.Errorf("expected DB_PASSWORD to override, got %q", cfg.DB.Password)
	}
}

func TestValidateHTTPMode(t *testing.T) {
	// http.mode 验证通过 YAML 文件，不通过环境变量
	cfg := validProdConfig()
	cfg.HTTP.Mode = "invalid"
	err := cfg.Validate("prod")
	if err == nil {
		t.Fatal("expected error for invalid http.mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid http.mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogLevel(t *testing.T) {
	cfg := validProdConfig()
	cfg.Log.Level = "trace"
	err := cfg.Validate("prod")
	if err == nil {
		t.Fatal("expected error for invalid log.level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log.level") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogFormat(t *testing.T) {
	cfg := validProdConfig()
	cfg.Log.Format = "xml"
	err := cfg.Validate("prod")
	if err == nil {
		t.Fatal("expected error for invalid log.format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log.format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func validProdConfig() Config {
	return Config{
		HTTP: HTTPConfig{
			Host:                 "0.0.0.0",
			Port:                 8080,
			Mode:                 HTTPModeRelease,
			ReadHeaderTimeoutSec: 5,
			ReadTimeoutSec:       15,
			WriteTimeoutSec:      30,
			IdleTimeoutSec:       60,
			ShutdownTimeoutSec:   30,
			MaxBodyBytes:         1 << 20,
			TrustedProxies:       []string{"127.0.0.1"},
			AllowedOrigins:       []string{"https://admin.example.com"},
		},
		Management: ManagementConfig{
			Host:            "127.0.0.1",
			Port:            9090,
			ProbeTimeoutSec: 2,
		},
		DB: DBConfig{
			Host:             "mariadb",
			Port:             3306,
			User:             "jimu",
			Password:         "strong-db-password",
			Database:         "jimu",
			MaxOpen:          20,
			MaxIdle:          5,
			MaxRetries:       5,
			RetryIntervalSec: 3,
		},
		Redis: RedisConfig{
			Addr:             "redis:6379",
			MaxRetries:       5,
			RetryIntervalSec: 3,
		},
		Log: LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatJSON,
			Output: "stdout",
		},
		Auth: AuthConfig{
			JWTSecret:             strings.Repeat("x", 32),
			Issuer:                "jimu",
			AccessExpireMin:       30,
			RefreshExpireDay:      7,
			LoginRateLimit:        10,
			LoginRateWindowSec:    60,
			RegisterRateLimit:     5,
			RegisterRateWindowSec: 300,
			ResetCodeTTLMin:       15,
		},
		Server: ServerConfig{
			TimeoutSec:     30,
			RateLimitRate:  100,
			RateLimitBurst: 200,
		},
	}
}

func TestValidateProdRejectsInsecureValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		key    string
	}{
		{"default JWT secret", func(c *Config) { c.Auth.JWTSecret = "change-me-in-production" }, "auth.jwt_secret"},
		{"short JWT secret", func(c *Config) { c.Auth.JWTSecret = "short" }, "auth.jwt_secret"},
		{"default DB password", func(c *Config) { c.DB.Password = "root" }, "db.password"},
		{"invalid management port", func(c *Config) { c.Management.Port = 0 }, "management.port"},
		{"wildcard CORS", func(c *Config) { c.HTTP.AllowedOrigins = []string{"*"} }, "http.allowed_origins"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProdConfig()
			tt.mutate(&cfg)
			err := cfg.Validate("prod")
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Validate() error = %v, want key %q", err, tt.key)
			}
			if strings.Contains(err.Error(), cfg.Auth.JWTSecret) || strings.Contains(err.Error(), cfg.DB.Password) {
				t.Fatalf("validation error leaked a secret: %v", err)
			}
		})
	}
}

func TestValidateRedisMode(t *testing.T) {
	cfg := validProdConfig()
	cfg.Redis.Mode = "unknown"
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "invalid redis.mode") {
		t.Fatalf("Validate() error = %v, want invalid redis.mode", err)
	}
}

func TestValidateRedisSentinelRequiresConfig(t *testing.T) {
	cfg := validProdConfig()
	cfg.Redis.Mode = RedisModeSentinel
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "sentinel") {
		t.Fatalf("Validate() error = %v, want sentinel config required", err)
	}

	cfg.Redis.MasterName = "mymaster"
	cfg.Redis.SentinelAddrs = []string{"127.0.0.1:26379"}
	if err := cfg.Validate("prod"); err != nil {
		t.Fatalf("Validate() with valid sentinel config, unexpected error: %v", err)
	}
}

func TestValidateRedisClusterRequiresAddrs(t *testing.T) {
	cfg := validProdConfig()
	cfg.Redis.Mode = RedisModeCluster
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "cluster") {
		t.Fatalf("Validate() error = %v, want cluster_addrs required", err)
	}

	cfg.Redis.ClusterAddrs = []string{"127.0.0.1:7000", "127.0.0.1:7001"}
	if err := cfg.Validate("prod"); err != nil {
		t.Fatalf("Validate() with valid cluster config, unexpected error: %v", err)
	}
}

func TestValidateRedisEmptyModeDefaultsSingle(t *testing.T) {
	cfg := validProdConfig()
	cfg.Redis.Mode = ""
	if err := cfg.Validate("prod"); err != nil {
		t.Fatalf("Validate() with empty redis.mode should default to single, got: %v", err)
	}
	if cfg.Redis.Mode != RedisModeSingle {
		t.Fatalf("empty redis.mode should default to %q, got %q", RedisModeSingle, cfg.Redis.Mode)
	}
}

func validProvisioningConfig() ProvisioningConfig {
	return ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []ProvisionRoleTemplate{
			{
				Name:        "管理员",
				Description: "租户管理员",
				Permissions: []ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
			},
		},
	}
}

func TestValidateProvisioningDisabled(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.Provisioning = ProvisioningConfig{Enabled: false}
	if err := cfg.Validate("prod"); err != nil {
		t.Fatalf("disabled provisioning should pass validation, got: %v", err)
	}
}

func TestValidateProvisioningValid(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.PublicRegistration = true
	cfg.Auth.Provisioning = validProvisioningConfig()
	if err := cfg.Validate("prod"); err != nil {
		t.Fatalf("valid provisioning should pass validation, got: %v", err)
	}
}

func TestValidateProvisioningRequiresPublicRegistration(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.Provisioning = validProvisioningConfig()
	cfg.Auth.PublicRegistration = false
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "auth.public_registration") {
		t.Fatalf("provisioning without public registration should fail, got: %v", err)
	}
}

func TestValidateProvisioningRequiresRoles(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.PublicRegistration = true
	cfg.Auth.Provisioning = ProvisioningConfig{Enabled: true}
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "auth.provisioning.roles") {
		t.Fatalf("provisioning without roles should fail, got: %v", err)
	}
}

func TestValidateProvisioningRejectsDuplicateRoleName(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.PublicRegistration = true
	p := validProvisioningConfig()
	p.Roles = append(p.Roles, p.Roles[0])
	cfg.Auth.Provisioning = p
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate template role name should fail, got: %v", err)
	}
}

func TestValidateProvisioningRejectsUnknownOwnerRole(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.PublicRegistration = true
	p := validProvisioningConfig()
	p.OwnerRole = "不存在"
	cfg.Auth.Provisioning = p
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "owner_role") {
		t.Fatalf("unknown owner_role should fail, got: %v", err)
	}
}

func TestValidateProvisioningRejectsIncompletePermission(t *testing.T) {
	cfg := validProdConfig()
	cfg.Auth.PublicRegistration = true
	p := validProvisioningConfig()
	p.Roles[0].Permissions = []ProvisionPermission{{Resource: "/api/v1/users"}}
	cfg.Auth.Provisioning = p
	err := cfg.Validate("prod")
	if err == nil || !strings.Contains(err.Error(), "resource and action") {
		t.Fatalf("permission without action should fail, got: %v", err)
	}
}

func TestValidateIPAllowlist(t *testing.T) {
	base, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	base.Security.IPAllowlist = []string{"10.0.0.0/8", "127.0.0.1"}
	if err := base.Validate("dev"); err != nil {
		t.Fatalf("valid allowlist rejected: %v", err)
	}

	base.Security.AdminIPAllowlist = []string{"not-a-cidr"}
	if err := base.Validate("dev"); err == nil {
		t.Fatal("invalid admin allowlist should be rejected")
	}
}

func TestValidateOAuthProviders(t *testing.T) {
	tests := []struct {
		name    string
		cfg     OAuthProviderConfig
		wantErr bool
	}{
		{"未启用时忽略空配置", OAuthProviderConfig{Enabled: false}, false},
		{"启用但缺 client_id", OAuthProviderConfig{Enabled: true, RedirectURL: "https://x/cb"}, true},
		{"启用但缺 redirect_url", OAuthProviderConfig{Enabled: true, ClientID: "id"}, true},
		{"内置提供商合法", OAuthProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb"}, false},
		{"OIDC issuer 合法", OAuthProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb", IssuerURL: "https://idp.example.com/realms/acme"}, false},
		{"OIDC issuer 非绝对地址", OAuthProviderConfig{Enabled: true, ClientID: "id", RedirectURL: "https://x/cb", IssuerURL: "idp.example.com"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOAuthProviders(OAuthConfig{Providers: map[string]OAuthProviderConfig{"p": tt.cfg}})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCapabilitiesConfigFieldMapping(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	conf := `
capabilities:
  enabled:
    - user
    - auth
`
	if err := v.ReadConfig(strings.NewReader(conf)); err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.Equal(t, []string{"user", "auth"}, cfg.Capabilities.Enabled)
}

func TestValidateCapabilitiesRejectsBlankName(t *testing.T) {
	cfg := minimalValidConfig(t)
	cfg.Capabilities.Enabled = []string{"user", " "}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected error for blank capability name")
	}
}

func TestValidateCapabilitiesRejectsDuplicate(t *testing.T) {
	cfg := minimalValidConfig(t)
	cfg.Capabilities.Enabled = []string{"user", "user"}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected error for duplicate capability name")
	}
}

func TestValidateCapabilitiesAllowsEmpty(t *testing.T) {
	cfg := minimalValidConfig(t)
	cfg.Capabilities.Enabled = nil
	if err := cfg.Validate("dev"); err != nil {
		t.Fatalf("empty enabled must be valid (means all), got %v", err)
	}
}

func TestValidateCapabilitiesTrimsAndDedupes(t *testing.T) {
	cfg := minimalValidConfig(t)
	cfg.Capabilities.Enabled = []string{"user", " user"}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected duplicate detection after trimming")
	}
	cfg.Capabilities.Enabled = []string{" user ", "auth"}
	if err := cfg.Validate("dev"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, []string{"user", "auth"}, cfg.Capabilities.Enabled)
}

// minimalValidConfig 返回一份通过 dev 校验的配置副本，供能力校验用例复用。
func minimalValidConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return *cfg
}
