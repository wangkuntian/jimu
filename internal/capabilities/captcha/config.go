package captcha

import (
	"errors"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "captcha"

// Config 验证码能力配置（原 config.CaptchaConfig，P2.1 下沉）。
type Config struct {
	Enabled bool `mapstructure:"enabled"` // 是否启用登录/注册验证码
	TTLMin  int  `mapstructure:"ttl_min"` // 验证码有效期（分钟）
}

// ApplyDefaults 本能力无配置层默认值（ttl_min 缺省时 Enabled 必须为 false，
// 否则由 Validate 拒绝），保留空实现以满足 SectionConfig 契约。
func (c *Config) ApplyDefaults() {}

// Validate 校验本能力配置段。仅在本能力**启用**时由组合根调用，
// 因此未启用能力的配置段既不出现也不校验（设计 §8）。
func (c Config) Validate() error {
	if c.Enabled && c.TTLMin <= 0 {
		return errors.New("captcha.ttl_min")
	}
	return nil
}

// Load 解码并校验本能力配置段。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
