package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
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
	HTTP   HTTPConfig   `mapstructure:"http"`
	DB     DBConfig     `mapstructure:"db"`
	Redis  RedisConfig  `mapstructure:"redis"`
	Log    LogConfig    `mapstructure:"log"`
	Auth   AuthConfig   `mapstructure:"auth"`
	Server ServerConfig `mapstructure:"server"`
	Cache  CacheConfig  `mapstructure:"cache"`
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

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	MaxOpen  int    `mapstructure:"max_open"`
	MaxIdle  int    `mapstructure:"max_idle"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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
	JWTSecret        string `mapstructure:"jwt_secret"`
	AccessExpireMin  int    `mapstructure:"access_expire_min"`
	RefreshExpireDay int    `mapstructure:"refresh_expire_day"`
}

// Load 加载配置
// 优先级：环境变量 > .env > app.{env}.yaml > app.yaml
func Load() (*Config, error) {
	// 加载 .env 文件（不报错如果文件不存在）
	_ = godotenv.Load()

	env := os.Getenv("JIMU_ENV")

	v := viper.New()
	v.SetConfigType("yaml")

	// 先加载基础配置 app.yaml
	v.SetConfigName("app")

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

	// 展开配置中的环境变量占位符 ${VAR}
	expandConfig(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// viper 的 AutomaticEnv 在 Unmarshal 时不生效，手动覆盖
	applyEnvOverrides(&cfg)

	// 校验枚举值
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyEnvOverrides 手动应用环境变量覆盖（viper AutomaticEnv 对 Unmarshal 不生效）
func applyEnvOverrides(cfg *Config) {
	// HTTP
	if v := os.Getenv("JIMU__HTTP__PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.HTTP.Port = p
		}
	}
	if v := os.Getenv("JIMU__HTTP__HOST"); v != "" {
		cfg.HTTP.Host = v
	}
	if v := os.Getenv("JIMU__HTTP__MODE"); v != "" {
		cfg.HTTP.Mode = v
	}
	// DB
	if v := os.Getenv("JIMU__DB__HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("JIMU__DB__PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = p
		}
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
	if v := os.Getenv("JIMU__REDIS__DB"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = d
		}
	}
	// Auth
	if v := os.Getenv("JIMU__AUTH__JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	// Log
	if v := os.Getenv("JIMU__LOG__LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("JIMU__LOG__FORMAT"); v != "" {
		cfg.Log.Format = v
	}
}

// expandConfig 展开配置值中的 ${VAR} 占位符
func expandConfig(v *viper.Viper) {
	for _, key := range v.AllKeys() {
		val := v.GetString(key)
		if strings.Contains(val, "${") {
			expanded := os.ExpandEnv(val)
			v.Set(key, expanded)
		}
	}
}

func (c *Config) validate() error {
	if !contains(validHTTPModes, c.HTTP.Mode) {
		return fmt.Errorf("invalid http.mode: %q, must be one of %v", c.HTTP.Mode, validHTTPModes)
	}
	if !contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid log.level: %q, must be one of %v", c.Log.Level, validLogLevels)
	}
	if !contains(validLogFormats, c.Log.Format) {
		return fmt.Errorf("invalid log.format: %q, must be one of %v", c.Log.Format, validLogFormats)
	}
	return nil
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
