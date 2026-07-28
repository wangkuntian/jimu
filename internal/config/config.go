package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	HTTP  HTTPConfig  `mapstructure:"http"`
	DB    DBConfig    `mapstructure:"db"`
	Redis RedisConfig `mapstructure:"redis"`
	Log   LogConfig   `mapstructure:"log"`
	Auth  AuthConfig  `mapstructure:"auth"`
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

// Load 加载配置，支持多环境
// 优先级：环境变量 > app.{env}.yaml > app.yaml
func Load() (*Config, error) {
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

	// viper Unmarshal 不自动应用 AutomaticEnv，手动覆盖
	applyEnvOverrides(&cfg)

	// 校验枚举值
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
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

func applyEnvOverrides(cfg *Config) {
	if port := os.Getenv("JIMU__HTTP__PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.HTTP.Port = p
		}
	}
	if host := os.Getenv("JIMU__HTTP__HOST"); host != "" {
		cfg.HTTP.Host = host
	}
	if mode := os.Getenv("JIMU__HTTP__MODE"); mode != "" {
		cfg.HTTP.Mode = mode
	}
	if host := os.Getenv("JIMU__DB__HOST"); host != "" {
		cfg.DB.Host = host
	}
	if port := os.Getenv("JIMU__DB__PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.DB.Port = p
		}
	}
	if secret := os.Getenv("JIMU__AUTH__JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if level := os.Getenv("JIMU__LOG__LEVEL"); level != "" {
		cfg.Log.Level = level
	}
	if format := os.Getenv("JIMU__LOG__FORMAT"); format != "" {
		cfg.Log.Format = format
	}
}
