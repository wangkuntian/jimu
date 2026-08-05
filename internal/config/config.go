package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"jimu/internal/platform/observability"

	"github.com/spf13/viper"
)

// 枚举常量
const (
	HTTPModeDebug   = "debug"
	HTTPModeRelease = "release"
	HTTPModeTest    = "test"

	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	LogFormatJSON    = "json"
	LogFormatConsole = "console"
)

var (
	validHTTPModes  = []string{HTTPModeDebug, HTTPModeRelease, HTTPModeTest}
	validLogLevels  = []string{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
	validLogFormats = []string{LogFormatJSON, LogFormatConsole}
)

type Config struct {
	HTTP       HTTPConfig                  `mapstructure:"http"`
	Management ManagementConfig            `mapstructure:"management"`
	DB         DBConfig                    `mapstructure:"db"`
	Redis      RedisConfig                 `mapstructure:"redis"`
	Log        LogConfig                   `mapstructure:"log"`
	Auth       AuthConfig                  `mapstructure:"auth"`
	Server     ServerConfig                `mapstructure:"server"`
	Cache      CacheConfig                 `mapstructure:"cache"`
	Audit      AuditConfig                 `mapstructure:"audit"`
	OTEL       observability.TracingConfig `mapstructure:"otel"`
}

// ServerConfig 服务运行时配置
type ServerConfig struct {
	TimeoutSec     int `mapstructure:"timeout_sec"`      // 请求超时秒数，0 表示不限制
	RateLimitRate  int `mapstructure:"rate_limit_rate"`  // 全局限流速率（每秒请求数），0 表示不限流
	RateLimitBurst int `mapstructure:"rate_limit_burst"` // 限流桶容量，允许的突发请求数
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Prefix string `mapstructure:"prefix"` // 缓存 key 前缀，用于多实例隔离
}

type ManagementConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	EnablePprof     bool   `mapstructure:"enable_pprof"`
	ProbeTimeoutSec int    `mapstructure:"probe_timeout_sec"`
}

type AuditConfig struct {
	QueueSize       int `mapstructure:"queue_size"`
	BatchSize       int `mapstructure:"batch_size"`
	FlushIntervalMS int `mapstructure:"flush_interval_ms"`
}

type HTTPConfig struct {
	Host                 string   `mapstructure:"host"`
	Port                 int      `mapstructure:"port"`
	Mode                 string   `mapstructure:"mode"`
	ReadHeaderTimeoutSec int      `mapstructure:"read_header_timeout_sec"`
	ReadTimeoutSec       int      `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec      int      `mapstructure:"write_timeout_sec"`
	IdleTimeoutSec       int      `mapstructure:"idle_timeout_sec"`
	ShutdownTimeoutSec   int      `mapstructure:"shutdown_timeout_sec"`
	MaxBodyBytes         int64    `mapstructure:"max_body_bytes"`
	TrustedProxies       []string `mapstructure:"trusted_proxies"`
	AllowedOrigins       []string `mapstructure:"allowed_origins"`
}

type DBConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	Database           string `mapstructure:"database"`
	MaxOpen            int    `mapstructure:"max_open"`
	MaxIdle            int    `mapstructure:"max_idle"`
	ConnMaxLifetimeSec int    `mapstructure:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec int    `mapstructure:"conn_max_idle_time_sec"`
	MaxRetries         int    `mapstructure:"max_retries"`
	RetryIntervalSec   int    `mapstructure:"retry_interval_sec"`
}

type RedisConfig struct {
	Addr             string `mapstructure:"addr"`
	Password         string `mapstructure:"password"`
	DB               int    `mapstructure:"db"`
	PoolSize         int    `mapstructure:"pool_size"`
	MinIdleConns     int    `mapstructure:"min_idle_conns"`
	ReadTimeoutSec   int    `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec  int    `mapstructure:"write_timeout_sec"`
	MaxRetries       int    `mapstructure:"max_retries"`
	RetryIntervalSec int    `mapstructure:"retry_interval_sec"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留文件数
	MaxAge     int    `mapstructure:"max_age"`     // 保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
}

type AuthConfig struct {
	JWTSecret             string `mapstructure:"jwt_secret"`
	Issuer                string `mapstructure:"issuer"`
	AccessExpireMin       int    `mapstructure:"access_expire_min"`
	RefreshExpireDay      int    `mapstructure:"refresh_expire_day"`
	PublicRegistration    bool   `mapstructure:"public_registration"`
	LoginRateLimit        int    `mapstructure:"login_rate_limit"`
	LoginRateWindowSec    int    `mapstructure:"login_rate_window_sec"`
	RegisterRateLimit     int    `mapstructure:"register_rate_limit"`
	RegisterRateWindowSec int    `mapstructure:"register_rate_window_sec"`
}

// Load 加载配置
// 优先级：环境变量 > .env > app.{env}.yaml > app.yaml
func Load() (*Config, error) {
	env := os.Getenv("JIMU_ENV")

	v := viper.New()

	// 注意：.env 文件由 docker-compose 的 env_file 注入为环境变量
	// 不需要 viper 加载 .env，避免与 docker-compose 冲突

	// 加载 yaml 配置
	v.SetConfigName("app")
	v.SetConfigType("yaml")

	// 查找项目根目录下的 configs/
	wd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		cfgDir := filepath.Join(wd, "configs")
		if _, err := os.Stat(cfgDir); err == nil {
			v.AddConfigPath(cfgDir)
			break
		}
		wd = filepath.Dir(wd)
	}
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read app.yaml: %w", err)
	}

	// 环境配置覆盖（如 app.prod.yaml）
	if env != "" && env != "dev" {
		v.SetConfigName("app." + env)
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to merge app.%s.yaml: %w", env, err)
			}
		}
	}

	// 环境变量覆盖：JIMU__HTTP__PORT=9090
	v.SetEnvPrefix("JIMU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// viper AutomaticEnv 对嵌套键的 Unmarshal 不生效，手动覆盖
	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(env); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyEnvOverrides 手动应用环境变量覆盖
func applyEnvOverrides(cfg *Config) error {
	// HTTP
	if err := overrideInt("JIMU__HTTP__PORT", &cfg.HTTP.Port); err != nil {
		return err
	}
	if v := os.Getenv("JIMU__HTTP__HOST"); v != "" {
		cfg.HTTP.Host = v
	}
	if v := os.Getenv("JIMU__HTTP__MODE"); v != "" {
		cfg.HTTP.Mode = v
	}
	for _, item := range []struct {
		key string
		dst *int
	}{
		{"JIMU__HTTP__READ_HEADER_TIMEOUT_SEC", &cfg.HTTP.ReadHeaderTimeoutSec},
		{"JIMU__HTTP__READ_TIMEOUT_SEC", &cfg.HTTP.ReadTimeoutSec},
		{"JIMU__HTTP__WRITE_TIMEOUT_SEC", &cfg.HTTP.WriteTimeoutSec},
		{"JIMU__HTTP__IDLE_TIMEOUT_SEC", &cfg.HTTP.IdleTimeoutSec},
		{"JIMU__HTTP__SHUTDOWN_TIMEOUT_SEC", &cfg.HTTP.ShutdownTimeoutSec},
	} {
		if err := overrideInt(item.key, item.dst); err != nil {
			return err
		}
	}
	if err := overrideInt64("JIMU__HTTP__MAX_BODY_BYTES", &cfg.HTTP.MaxBodyBytes); err != nil {
		return err
	}
	if v := os.Getenv("JIMU__HTTP__TRUSTED_PROXIES"); v != "" {
		cfg.HTTP.TrustedProxies = splitList(v)
	}
	if v := os.Getenv("JIMU__HTTP__ALLOWED_ORIGINS"); v != "" {
		cfg.HTTP.AllowedOrigins = splitList(v)
	}
	// Management
	if v := os.Getenv("JIMU__MANAGEMENT__HOST"); v != "" {
		cfg.Management.Host = v
	}
	if err := overrideInt("JIMU__MANAGEMENT__PORT", &cfg.Management.Port); err != nil {
		return err
	}
	if err := overrideBool("JIMU__MANAGEMENT__ENABLE_PPROF", &cfg.Management.EnablePprof); err != nil {
		return err
	}
	if err := overrideInt("JIMU__MANAGEMENT__PROBE_TIMEOUT_SEC", &cfg.Management.ProbeTimeoutSec); err != nil {
		return err
	}
	// DB
	if v := os.Getenv("JIMU__DB__HOST"); v != "" {
		cfg.DB.Host = v
	}
	if err := overrideInt("JIMU__DB__PORT", &cfg.DB.Port); err != nil {
		return err
	}
	if v := os.Getenv("JIMU__DB__USER"); v != "" {
		cfg.DB.User = v
	}
	if v := os.Getenv("JIMU__DB__PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("JIMU__DB__DATABASE"); v != "" {
		cfg.DB.Database = v
	}
	// Redis
	if v := os.Getenv("JIMU__REDIS__ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("JIMU__REDIS__PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if err := overrideInt("JIMU__REDIS__DB", &cfg.Redis.DB); err != nil {
		return err
	}
	for _, item := range []struct {
		key string
		dst *int
	}{
		{"JIMU__DB__CONN_MAX_LIFETIME_SEC", &cfg.DB.ConnMaxLifetimeSec},
		{"JIMU__DB__CONN_MAX_IDLE_TIME_SEC", &cfg.DB.ConnMaxIdleTimeSec},
		{"JIMU__DB__MAX_RETRIES", &cfg.DB.MaxRetries},
		{"JIMU__DB__RETRY_INTERVAL_SEC", &cfg.DB.RetryIntervalSec},
		{"JIMU__REDIS__POOL_SIZE", &cfg.Redis.PoolSize},
		{"JIMU__REDIS__MIN_IDLE_CONNS", &cfg.Redis.MinIdleConns},
		{"JIMU__REDIS__READ_TIMEOUT_SEC", &cfg.Redis.ReadTimeoutSec},
		{"JIMU__REDIS__WRITE_TIMEOUT_SEC", &cfg.Redis.WriteTimeoutSec},
		{"JIMU__REDIS__MAX_RETRIES", &cfg.Redis.MaxRetries},
		{"JIMU__REDIS__RETRY_INTERVAL_SEC", &cfg.Redis.RetryIntervalSec},
	} {
		if err := overrideInt(item.key, item.dst); err != nil {
			return err
		}
	}
	// Auth
	if v := os.Getenv("JIMU__AUTH__JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("JIMU__AUTH__ISSUER"); v != "" {
		cfg.Auth.Issuer = v
	}
	if err := overrideBool("JIMU__AUTH__PUBLIC_REGISTRATION", &cfg.Auth.PublicRegistration); err != nil {
		return err
	}
	for _, item := range []struct {
		key string
		dst *int
	}{
		{"JIMU__AUTH__ACCESS_EXPIRE_MIN", &cfg.Auth.AccessExpireMin},
		{"JIMU__AUTH__REFRESH_EXPIRE_DAY", &cfg.Auth.RefreshExpireDay},
		{"JIMU__AUTH__LOGIN_RATE_LIMIT", &cfg.Auth.LoginRateLimit},
		{"JIMU__AUTH__LOGIN_RATE_WINDOW_SEC", &cfg.Auth.LoginRateWindowSec},
		{"JIMU__AUTH__REGISTER_RATE_LIMIT", &cfg.Auth.RegisterRateLimit},
		{"JIMU__AUTH__REGISTER_RATE_WINDOW_SEC", &cfg.Auth.RegisterRateWindowSec},
	} {
		if err := overrideInt(item.key, item.dst); err != nil {
			return err
		}
	}
	// Log
	if v := os.Getenv("JIMU__LOG__LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("JIMU__LOG__FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	// Audit
	for _, item := range []struct {
		key string
		dst *int
	}{
		{"JIMU__AUDIT__QUEUE_SIZE", &cfg.Audit.QueueSize},
		{"JIMU__AUDIT__BATCH_SIZE", &cfg.Audit.BatchSize},
		{"JIMU__AUDIT__FLUSH_INTERVAL_MS", &cfg.Audit.FlushIntervalMS},
	} {
		if err := overrideInt(item.key, item.dst); err != nil {
			return err
		}
	}
	// OTEL
	if v := os.Getenv("JIMU__OTEL__ENABLED"); v != "" {
		if err := overrideBool("JIMU__OTEL__ENABLED", &cfg.OTEL.Enabled); err != nil {
			return err
		}
	}
	if v := os.Getenv("JIMU__OTEL__ENDPOINT"); v != "" {
		cfg.OTEL.Endpoint = v
	}
	if v := os.Getenv("JIMU__OTEL__SERVICE_NAME"); v != "" {
		cfg.OTEL.ServiceName = v
	}
	if v := os.Getenv("JIMU__OTEL__SERVICE_VERSION"); v != "" {
		cfg.OTEL.ServiceVersion = v
	}
	if v := os.Getenv("JIMU__OTEL__SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.OTEL.SampleRate = f
		}
	}
	return nil
}

func overrideInt(key string, dst *int) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("invalid %s", strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(key, "JIMU__"), "__", ".")))
	}
	*dst = n
	return nil
}

func overrideInt64(key string, dst *int64) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid %s", strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(key, "JIMU__"), "__", ".")))
	}
	*dst = n
	return nil
}

func overrideBool(key string, dst *bool) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("invalid %s", strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(key, "JIMU__"), "__", ".")))
	}
	*dst = b
	return nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
