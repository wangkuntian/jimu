package audit

import (
	"errors"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "audit"

// Config 审计能力配置（原 config.AuditConfig，P2.1 下沉）。
type Config struct {
	QueueSize       int    `mapstructure:"queue_size"`
	BatchSize       int    `mapstructure:"batch_size"`
	FlushIntervalMS int    `mapstructure:"flush_interval_ms"`
	HashSecret      string `mapstructure:"hash_secret"` // 审计链 HMAC 密钥；为空时退化为 SHA-256
}

// ApplyDefaults 应用本能力的环境变量覆盖（无配置层默认值：各字段必填，由 Validate 兜底）。
// 段的环境覆盖随段一起归属能力，语义与内核段一致：环境变量 > YAML。
func (c *Config) ApplyDefaults() {
	// 审计链 HMAC 密钥：配置后篡改者无法重算整条链；支持 _FILE 形式（Docker Secrets 兼容）
	if v := config.GetEnvOrFile("AUDIT_HASH_SECRET_FILE", "AUDIT_HASH_SECRET"); v != "" {
		c.HashSecret = v
	}
}

// Validate 校验本能力配置段。仅在本能力启用时由组合根调用（设计 §8）。
func (c Config) Validate() error {
	if c.QueueSize <= 0 || c.BatchSize <= 0 || c.BatchSize > c.QueueSize || c.FlushIntervalMS <= 0 {
		return errors.New("audit configuration")
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
