package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
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

func TestStorageConfigFieldMapping(t *testing.T) {
	// 验证 storage 的 S3/OSS/MinIO 字段可从 YAML 映射到结构体
	// （直接驱动 viper，避免依赖项目 configs/ 目录的实际值）
	v := viper.New()
	v.SetConfigType("yaml")
	conf := `
storage:
  type: "oss"
  endpoint: "oss-cn-hangzhou.aliyuncs.com"
  region: "cn-hangzhou"
  bucket: "my-bucket"
  access_key: "ak"
  secret_key: "sk"
  path_style: true
`
	if err := v.ReadConfig(strings.NewReader(conf)); err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Storage.Type != "oss" || cfg.Storage.Endpoint != "oss-cn-hangzhou.aliyuncs.com" {
		t.Errorf("storage type/endpoint not mapped: %+v", cfg.Storage)
	}
	if cfg.Storage.Bucket != "my-bucket" || !cfg.Storage.PathStyle || cfg.Storage.Region != "cn-hangzhou" {
		t.Errorf("storage bucket/region/path_style not mapped: %+v", cfg.Storage)
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

func TestValidateOutboxPublisher(t *testing.T) {
	cfg := validProdConfig()
	cfg.Outbox.Publisher = "invalid"
	err := cfg.Validate("prod")
	if err == nil {
		t.Fatal("expected error for invalid outbox.publisher, got nil")
	}
	if !strings.Contains(err.Error(), "invalid outbox.publisher") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOutboxMQWithRedisQueue(t *testing.T) {
	cfg := validProdConfig()
	cfg.Outbox.Publisher = OutboxPublisherMQ
	cfg.Queue.Type = QueueTypeRedis
	if err := cfg.Validate("prod"); err != nil {
		t.Errorf("mq + redis should be valid, got: %v", err)
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

func TestValidateSchedulerStore(t *testing.T) {
	cfg := validProdConfig()
	cfg.Scheduler.Store = "etcd"
	err := cfg.Validate("prod")
	if err == nil {
		t.Fatal("expected error for invalid scheduler.store, got nil")
	}
	if !strings.Contains(err.Error(), "invalid scheduler.store") {
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
		Queue: QueueConfig{
			Type: QueueTypeRedis,
		},
		Outbox: OutboxConfig{
			Publisher: OutboxPublisherEventBus,
		},
		Scheduler: SchedulerConfig{
			Store: SchedulerStoreMemory,
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
