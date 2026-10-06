package outbox

import (
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDecoder struct {
	values map[string]any
}

func (d *fakeDecoder) UnmarshalKey(key string, rawVal any) error {
	if p, ok := rawVal.(*Config); ok {
		if v, ok := d.values[key].(Config); ok {
			*p = v
		}
	}
	return nil
}

func TestConfigKeyIsStable(t *testing.T) {
	assert.Equal(t, "outbox", ConfigKey, "对外配置键不得变化")
}

// TestDescriptorDeclaresConfigSection 描述符必须声明本段，否则框架不会加载/校验它。
func TestDescriptorDeclaresConfigSection(t *testing.T) {
	for _, spec := range Descriptor.Configs {
		if spec.Section == ConfigKey {
			require.NotNil(t, spec.New)
			_, ok := spec.New().(config.SectionConfig)
			require.True(t, ok, "段实例必须实现 config.SectionConfig")
			return
		}
	}
	t.Fatalf("Descriptor 必须声明配置段 %q", ConfigKey)
}

// loadConfig 走框架同款机制（config.LoadSection：解码 → 默认值 → 校验）。
func loadConfig(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// TestValidateOutboxPublisher 迁移自 internal/config 的同名用例。
func TestValidateOutboxPublisher(t *testing.T) {
	require.NoError(t, Config{Publisher: PublisherEventBus}.Validate())
	require.NoError(t, Config{Publisher: PublisherMQ}.Validate())

	_, err := loadConfig(&fakeDecoder{values: map[string]any{"outbox": Config{Publisher: "invalid"}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outbox.publisher")
}

func TestOutboxRetentionConfigValidation(t *testing.T) {
	valid := Config{Publisher: PublisherEventBus, Retention: RetentionConfig{Enabled: true, Cron: "30 3 * * *", BatchSize: 500, EventDays: 7}}
	require.NoError(t, valid.Validate())
	for name, retention := range map[string]RetentionConfig{
		"missing cron":   {Enabled: true, BatchSize: 500, EventDays: 7},
		"negative batch": {Enabled: true, Cron: "30 3 * * *", BatchSize: -1, EventDays: 7},
		"negative days":  {Enabled: true, Cron: "30 3 * * *", BatchSize: 500, EventDays: -1},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			cfg.Retention = retention
			require.Error(t, cfg.Validate())
		})
	}
}

func TestOutboxRetentionRulesOwnPublishedEvents(t *testing.T) {
	rules := outboxRetentionRules(RetentionConfig{EventDays: 7})
	require.Len(t, rules, 1)
	assert.Equal(t, "outbox_events", rules[0].Table)
	assert.Equal(t, "published_at", rules[0].TimeColumn)
	assert.Equal(t, "published_at IS NOT NULL", rules[0].Condition)
	assert.Equal(t, 7, rules[0].Days)
}

func TestUsesMQ(t *testing.T) {
	assert.True(t, Config{Publisher: PublisherMQ}.UsesMQ())
	assert.False(t, Config{Publisher: PublisherEventBus}.UsesMQ())
}

func TestLoadMissingSectionRejected(t *testing.T) {
	// publisher 必填：段缺失时零值非法（与下沉前一致）
	_, err := loadConfig(&fakeDecoder{})
	require.Error(t, err)
}
