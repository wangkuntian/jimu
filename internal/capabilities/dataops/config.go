package dataops

import (
	"errors"
	"strings"

	"jimu/internal/config"
)

const ConfigKey = "dataops"

type Config struct {
	Retention RetentionConfig `mapstructure:"retention"`
}

type RetentionConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Cron          string `mapstructure:"cron"`
	BatchSize     int    `mapstructure:"batch_size"`
	ImportJobDays int    `mapstructure:"import_job_days"`
}

func (c *Config) ApplyDefaults() {}

func (c Config) Validate() error {
	if c.Retention.Enabled && strings.TrimSpace(c.Retention.Cron) == "" {
		return errors.New("dataops.retention.cron is required when enabled")
	}
	if c.Retention.BatchSize < 0 || c.Retention.ImportJobDays < 0 {
		return errors.New("dataops.retention values must not be negative")
	}
	return nil
}

var _ config.SectionConfig = (*Config)(nil)
