package mfa

import (
	"strings"
	"testing"

	"jimu/internal/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mfaConfigDecoder struct{ v *viper.Viper }

func (d mfaConfigDecoder) UnmarshalKey(key string, out any) error { return d.v.UnmarshalKey(key, out) }

func TestMFAConfigDescriptorAndRetention(t *testing.T) {
	assert.Equal(t, "mfa", ConfigKey)
	for _, spec := range Descriptor.Configs {
		if spec.Section == ConfigKey {
			_, ok := spec.New().(config.SectionConfig)
			require.True(t, ok)
			break
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader("mfa:\n  trusted_device_days: 30\n  retention:\n    enabled: false\n    cron: '30 3 * * *'\n    batch_size: 500\n    expired_device_days: 7\n")))
	settings := &Settings{}
	require.NoError(t, config.LoadSection(mfaConfigDecoder{v: v}, ConfigKey, settings))
	assert.False(t, settings.Retention.Enabled)
	assert.Equal(t, 30, settings.TrustedDeviceDays)
	assert.Equal(t, 7, settings.Retention.ExpiredDeviceDays)
}

func TestMFAConfigRejectsNegativeRetention(t *testing.T) {
	settings := Settings{TrustedDeviceDays: 30, Retention: RetentionConfig{Enabled: true, Cron: "30 3 * * *", ExpiredDeviceDays: -1}}
	require.Error(t, settings.Validate())
	settings.Retention.ExpiredDeviceDays = 7
	settings.TrustedDeviceDays = -1
	require.Error(t, settings.Validate())
}
