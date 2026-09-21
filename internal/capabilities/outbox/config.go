package outbox

import (
	"fmt"

	"jimu/internal/config"
)

// ConfigKey 本能力在 app.yaml 中的配置段键。
const ConfigKey = "outbox"

// 发布器类型取值。
const (
	PublisherEventBus = "event_bus"
	PublisherMQ       = "mq"
)

var validPublishers = []string{PublisherEventBus, PublisherMQ}

// Config Outbox 配置（原 config.OutboxConfig，P2.1 下沉）。
type Config struct {
	Publisher string `mapstructure:"publisher"` // 发布器类型：event_bus, mq
}

// ApplyDefaults 本能力无配置层默认值（publisher 必填，由 Validate 兜底）。
func (c *Config) ApplyDefaults() {}

// Validate 校验本能力配置段。仅在本能力启用时由组合根调用（设计 §8）。
// 「publisher=mq 时队列类型必须可承载」属跨能力约束，由组合根结合 queue 配置校验。
func (c Config) Validate() error {
	for _, p := range validPublishers {
		if c.Publisher == p {
			return nil
		}
	}
	return fmt.Errorf("outbox.publisher: %q, must be one of %v", c.Publisher, validPublishers)
}

// UsesMQ 报告是否配置为 MQ 投递。
func (c Config) UsesMQ() bool { return c.Publisher == PublisherMQ }

// Load 解码并校验本能力配置段。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
