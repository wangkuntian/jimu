package uploadsec

import (
	"time"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "upload"

// Config 上传安全能力配置段（形状与 YAML 键名同下沉前一致）。
type Config struct {
	ClamAV ClamAVSection `mapstructure:"clamav"`
}

// ClamAVSection ClamAV 扫描的配置段表示（超时以秒计，与 YAML 一致）。
type ClamAVSection struct {
	Enabled    bool   `mapstructure:"enabled"`     // 是否启用，false 时上传不扫描
	Address    string `mapstructure:"address"`     // clamd 监听地址，如 127.0.0.1:3310
	TimeoutSec int    `mapstructure:"timeout_sec"` // 扫描超时（秒），0 用默认 10
}

// Scanner 按配置构建病毒扫描器：未启用时返回 nil（上传不扫描，向后兼容）。
// YAML 秒级超时到扫描器 Duration 的换算收在本能力内，组合根只取结果。
func (c Config) Scanner() Scanner {
	if !c.ClamAV.Enabled {
		return nil
	}
	return NewClamAVScanner(ClamAVConfig{
		Address: c.ClamAV.Address,
		Timeout: time.Duration(c.ClamAV.TimeoutSec) * time.Second,
	})
}

// ApplyDefaults 本能力无配置层默认值：扫描超时/分片大小缺省值由
// NewClamAVScanner 填充，与下沉前一致。
func (c *Config) ApplyDefaults() {}

// Validate 本能力无配置校验。
func (c *Config) Validate() error { return nil }

// Load 解码本能力配置段。仅在本能力启用时由组合根调用（设计 §8）。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
