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
	HTTP        HTTPConfig                  `mapstructure:"http"`
	Management  ManagementConfig            `mapstructure:"management"`
	DB          DBConfig                    `mapstructure:"db"`
	Redis       RedisConfig                 `mapstructure:"redis"`
	Log         LogConfig                   `mapstructure:"log"`
	Auth        AuthConfig                  `mapstructure:"auth"`
	Server      ServerConfig                `mapstructure:"server"`
	Cache       CacheConfig                 `mapstructure:"cache"`
	Audit       AuditConfig                 `mapstructure:"audit"`
	Storage     StorageConfig               `mapstructure:"storage"`
	OTEL        observability.TracingConfig `mapstructure:"otel"`
	// 元数据（非 YAML 配置，运行时注入）
	Version     string `mapstructure:"-"`
	Environment string `mapstructure:"-"`
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

type StorageConfig struct {
	Type    string `mapstructure:"type"`     // local, s3, oss, minio
	BaseDir string `mapstructure:"base_dir"` // 本地存储目录
	BaseURL string `mapstructure:"base_url"` // 访问 URL 前缀
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
	Host               string   `mapstructure:"host"`
	Port               int      `mapstructure:"port"`
	User               string   `mapstructure:"user"`
	Password           string   `mapstructure:"password"`
	Database           string   `mapstructure:"database"`
	MaxOpen            int      `mapstructure:"max_open"`
	MaxIdle            int      `mapstructure:"max_idle"`
	ConnMaxLifetimeSec int      `mapstructure:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec int      `mapstructure:"conn_max_idle_time_sec"`
	MaxRetries         int      `mapstructure:"max_retries"`
	RetryIntervalSec   int      `mapstructure:"retry_interval_sec"`
	// 读写分离
	ReadHosts []string `mapstructure:"read_hosts"` // 从库地址列表
	ReadPorts []int    `mapstructure:"read_ports"` // 从库端口列表
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
	env := os.Getenv("APP_ENV")

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

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 敏感配置环境变量覆盖（不使用 JIMU_ 前缀）
	applyEnvOverrides(&cfg)

	if err := cfg.Validate(env); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyEnvOverrides 应用环境变量覆盖（简洁命名，无 JIMU_ 前缀）
// 支持 _FILE 后缀从文件读取敏感值（Docker Secrets 兼容）
func applyEnvOverrides(cfg *Config) {
	// 认证：优先 JWT_SECRET_FILE，其次 JWT_SECRET
	if v := getEnvOrFile("JWT_SECRET_FILE", "JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	// 数据库
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DB.User = v
	}
	// 密码：优先 DB_PASSWORD_FILE，其次 DB_PASSWORD
	if v := getEnvOrFile("DB_PASSWORD_FILE", "DB_PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DB.Database = v
	}
	// Redis
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := getEnvOrFile("REDIS_PASSWORD_FILE", "REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
}

// getEnvOrFile 优先从 _FILE 指向的文件读取，其次直接读取环境变量
func getEnvOrFile(fileKey, directKey string) string {
	// 优先从文件读取（Docker Secrets）
	if path := os.Getenv(fileKey); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			// 去除首尾空白（secret 文件通常有末尾换行符）
			return strings.TrimSpace(string(data))
		}
	}
	// 回退到直接环境变量
	return os.Getenv(directKey)
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
