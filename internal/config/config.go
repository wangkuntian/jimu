package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
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

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("app")
	v.SetConfigType("yaml")

	// Try to find project root by looking for configs/ directory
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

	// Environment variable override: JIMU__HTTP__PORT=9090
	v.SetEnvPrefix("JIMU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	// Bind specific env vars to ensure override works on Unmarshal
	for _, key := range []string{"http.port", "http.host", "http.mode", "db.host", "db.port",
		"db.user", "db.password", "db.database", "redis.addr", "log.level", "auth.jwt_secret"} {
		_ = v.BindEnv(key)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Apply env overrides (viper's Unmarshal doesn't apply AutomaticEnv for nested keys)
	applyEnvOverrides(&cfg)

	return &cfg, nil
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
}
