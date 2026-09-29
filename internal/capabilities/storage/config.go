package storage

import "jimu/internal/config"

// ConfigKey 本包在 app.yaml 中的配置段键。
const ConfigKey = "storage"

// ApplyDefaults 本包无配置层默认值：缺省值由 New 在构造各驱动时填充
// （如 local 的 base_dir 缺省为 "storage"），保持与下沉前完全一致。
func (c *Config) ApplyDefaults() {}

// Validate 本包无配置校验；非法驱动类型由 New 返回错误。
func (c *Config) Validate() error { return nil }

// Load 解码本包配置段。存储不属 catalog 能力，由组合根无条件加载。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
