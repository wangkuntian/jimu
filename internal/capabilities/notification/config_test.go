package notification

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type viperSection struct{ v *viper.Viper }

func (s viperSection) UnmarshalKey(key string, rawVal any) error {
	return s.v.UnmarshalKey(key, rawVal)
}

func yamlSection(t *testing.T, yaml string) viperSection {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(yaml)))
	return viperSection{v: v}
}

func TestConfigKeysAreStable(t *testing.T) {
	assert.Equal(t, "email", EmailKey, "对外配置键不得变化")
	assert.Equal(t, "sms", SMSKey, "对外配置键不得变化")
	assert.Equal(t, "notification", NotificationKey, "对外配置键不得变化")
}

// TestLoadMapsAllThreeSections YAML 键名与下沉前一致。
func TestLoadMapsAllThreeSections(t *testing.T) {
	cfg, err := Load(yamlSection(t, `
email:
  enabled: true
  host: "smtp.example.com"
  port: 587
  username: "u"
  password: "p"
  from: "noreply@example.com"
sms:
  enabled: true
  provider: "aliyun"
  api_key: "ak"
  api_secret: "sk"
  sign_name: "Jimu"
notification:
  webhook:
    sign_secret: "whsec"
`))
	require.NoError(t, err)

	assert.True(t, cfg.Email.Enabled)
	assert.Equal(t, "smtp.example.com", cfg.Email.Host)
	assert.Equal(t, 587, cfg.Email.Port)
	assert.Equal(t, "noreply@example.com", cfg.Email.From)

	assert.True(t, cfg.SMS.Enabled)
	assert.Equal(t, "aliyun", cfg.SMS.Provider)
	assert.Equal(t, "ak", cfg.SMS.APIKey)
	assert.Equal(t, "Jimu", cfg.SMS.SignName)

	assert.Equal(t, "whsec", cfg.Notification.Webhook.SignSecret)
}

// TestLoadMissingSectionsFallBackToLog 段缺失时渠道未启用（装配侧回退日志渠道）。
func TestLoadMissingSectionsFallBackToLog(t *testing.T) {
	cfg, err := Load(yamlSection(t, ""))
	require.NoError(t, err)
	assert.False(t, cfg.Email.Enabled)
	assert.False(t, cfg.SMS.Enabled)
	assert.Empty(t, cfg.Notification.Webhook.SignSecret)
}
