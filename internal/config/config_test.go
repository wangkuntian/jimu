package config

import (
	"strings"
	"testing"
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

func TestEnvOverride(t *testing.T) {
	t.Setenv("JIMU__HTTP__PORT", "9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.HTTP.Port)
	}
}

func TestValidateHTTPMode(t *testing.T) {
	t.Setenv("JIMU__HTTP__MODE", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid http.mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid http.mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogLevel(t *testing.T) {
	t.Setenv("JIMU__LOG__LEVEL", "trace")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log.level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log.level") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogFormat(t *testing.T) {
	t.Setenv("JIMU__LOG__FORMAT", "xml")

	_, err := Load()
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
			Host:     "mariadb",
			Port:     3306,
			User:     "jimu",
			Password: "strong-db-password",
			Database: "jimu",
			MaxOpen:  20,
			MaxIdle:  5,
		},
		Redis: RedisConfig{Addr: "redis:6379"},
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
		},
		Server: ServerConfig{
			TimeoutSec:     30,
			RateLimitRate:  100,
			RateLimitBurst: 200,
		},
		Audit: AuditConfig{
			QueueSize:       256,
			BatchSize:       50,
			FlushIntervalMS: 500,
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

func TestLoadOverridesNewNestedFields(t *testing.T) {
	t.Setenv("JIMU__MANAGEMENT__PORT", "9191")
	t.Setenv("JIMU__AUTH__PUBLIC_REGISTRATION", "true")
	t.Setenv("JIMU__HTTP__MAX_BODY_BYTES", "2048")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Management.Port != 9191 || !cfg.Auth.PublicRegistration || cfg.HTTP.MaxBodyBytes != 2048 {
		t.Fatalf("nested overrides not applied: %#v", cfg)
	}
}
