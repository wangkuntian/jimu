package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP       HTTPConfig       `mapstructure:"http" validate:"required"`
	Management ManagementConfig `mapstructure:"management" validate:"required"`
	Log        LogConfig        `mapstructure:"log" validate:"required"`
}

type HTTPConfig struct {
	Host                 string   `mapstructure:"host" validate:"required"`
	Port                 int      `mapstructure:"port" validate:"min=1,max=65535"`
	Mode                 string   `mapstructure:"mode" validate:"required,oneof=debug release test"`
	ReadHeaderTimeoutSec int      `mapstructure:"read_header_timeout_sec" validate:"min=1"`
	ReadTimeoutSec       int      `mapstructure:"read_timeout_sec" validate:"min=1"`
	WriteTimeoutSec      int      `mapstructure:"write_timeout_sec" validate:"min=1"`
	IdleTimeoutSec       int      `mapstructure:"idle_timeout_sec" validate:"min=1"`
	ShutdownTimeoutSec   int      `mapstructure:"shutdown_timeout_sec" validate:"min=1"`
	MaxBodyBytes         int64    `mapstructure:"max_body_bytes" validate:"min=1"`
	TrustedProxies       []string `mapstructure:"trusted_proxies"`
	AllowedOrigins       []string `mapstructure:"allowed_origins"`
}

type ManagementConfig struct {
	Host            string `mapstructure:"host" validate:"required"`
	Port            int    `mapstructure:"port" validate:"min=1,max=65535"`
	EnablePprof     bool   `mapstructure:"enable_pprof"`
	ProbeTimeoutSec int    `mapstructure:"probe_timeout_sec" validate:"min=1"`
}

type LogConfig struct {
	Level      string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format     string `mapstructure:"format" validate:"required,oneof=json console"`
	Output     string `mapstructure:"output" validate:"required"`
	MaxSize    int    `mapstructure:"max_size" validate:"min=1"`
	MaxBackups int    `mapstructure:"max_backups" validate:"min=1"`
	MaxAge     int    `mapstructure:"max_age" validate:"min=1"`
	Compress   bool   `mapstructure:"compress"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath())
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read %s: %w", configPath(), err)
	}
	var cfg Config
	if err := v.UnmarshalExact(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	applyEnvironment(&cfg)
	if err := validator.New().Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func configPath() string {
	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env != "" && env != "dev" {
		candidate := filepath.Join("configs", "app."+env+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("configs", "app.yaml")
}

func applyEnvironment(cfg *Config) {
	if value := os.Getenv("HTTP_HOST"); value != "" {
		cfg.HTTP.Host = value
	}
	if value := os.Getenv("HTTP_PORT"); value != "" {
		_, _ = fmt.Sscan(value, &cfg.HTTP.Port)
	}
	if value := os.Getenv("MANAGEMENT_HOST"); value != "" {
		cfg.Management.Host = value
	}
	if value := os.Getenv("MANAGEMENT_PORT"); value != "" {
		_, _ = fmt.Sscan(value, &cfg.Management.Port)
	}
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		cfg.Log.Level = value
	}
}
