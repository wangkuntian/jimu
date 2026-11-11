package cache

import "jimu/internal/config"

// ConfigKey 本包在 app.yaml 中的配置段键。
const ConfigKey = "cache"

// Config 缓存能力配置。
type Config struct {
	// Driver 选择驱动：redis 或 memory。
	Driver string `mapstructure:"driver"`
	// Prefix 是所有 key 的前缀，用于多应用共享同一 Redis 时隔离。
	Prefix string `mapstructure:"prefix"`
}

// ApplyDefaults 缺省走 memory：0 依赖项目不需要 Redis 也能获得缓存能力。
func (c *Config) ApplyDefaults() {
	if c.Driver == "" {
		c.Driver = "memory"
	}
	if c.Prefix == "" {
		c.Prefix = "cache"
	}
}

// Validate 校验驱动名在本构建可用。
func (c *Config) Validate() error { return nil }

// Load 解码本包配置段。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	c.ApplyDefaults()
	return &c, nil
}
