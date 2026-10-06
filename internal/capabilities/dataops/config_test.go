package dataops

import (
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configDecoder struct{ cfg Config }

func (d configDecoder) UnmarshalKey(key string, out any) error {
	if key == ConfigKey {
		*out.(*Config) = d.cfg
	}
	return nil
}

func TestDataopsDescriptorDeclaresConfig(t *testing.T) {
	assert.Equal(t, "dataops", ConfigKey)
	for _, spec := range Descriptor.Configs {
		if spec.Section == ConfigKey {
			_, ok := spec.New().(config.SectionConfig)
			require.True(t, ok)
			return
		}
	}
	t.Fatalf("Descriptor must declare %q", ConfigKey)
}

func TestDataopsRetentionConfigValidation(t *testing.T) {
	valid := Config{Retention: RetentionConfig{Enabled: true, Cron: "30 3 * * *", BatchSize: 500, ImportJobDays: 90}}
	require.NoError(t, valid.Validate())
	for name, retention := range map[string]RetentionConfig{
		"missing cron":   {Enabled: true, BatchSize: 500, ImportJobDays: 90},
		"negative batch": {Enabled: true, Cron: "30 3 * * *", BatchSize: -1, ImportJobDays: 90},
		"negative days":  {Enabled: true, Cron: "30 3 * * *", BatchSize: 500, ImportJobDays: -1},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			cfg.Retention = retention
			require.Error(t, cfg.Validate())
		})
	}

	loaded := &Config{}
	err := config.LoadSection[config.SectionConfig](configDecoder{cfg: Config{
		Retention: RetentionConfig{BatchSize: 500, ImportJobDays: 90},
	}}, ConfigKey, loaded)
	require.NoError(t, err)
	assert.False(t, loaded.Retention.Enabled)
	assert.Equal(t, 90, loaded.Retention.ImportJobDays)
}
