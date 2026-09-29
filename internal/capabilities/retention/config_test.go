package retention

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

func TestConfigKeyIsStable(t *testing.T) {
	assert.Equal(t, "retention", ConfigKey, "对外配置键不得变化")
}

// TestValidateOnlyWhenEnabled 校验条件与下沉前逐字一致：仅 enabled 时校验。
func TestValidateOnlyWhenEnabled(t *testing.T) {
	// 未启用：非法值也放行（原有语义）
	require.NoError(t, Config{Enabled: false, Cron: "", BatchSize: -1}.Validate())

	// 启用但缺 cron
	err := Config{Enabled: true, BatchSize: 500}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention.cron")

	// 启用但 batch_size 为负
	err = Config{Enabled: true, Cron: "30 3 * * *", BatchSize: -1}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention.batch_size")

	// 启用且合法
	require.NoError(t, Config{Enabled: true, Cron: "30 3 * * *", BatchSize: 500}.Validate())
}

// TestLoadMapsYAMLKeys YAML 键名与下沉前一致（含 trusted_device_days）。
func TestLoadMapsYAMLKeys(t *testing.T) {
	cfg, err := Load(yamlSection(t, `
retention:
  enabled: true
  cron: "30 3 * * *"
  batch_size: 200
  audit_log_days: 180
  job_days: 7
  job_history_days: 30
  dead_letter_days: 30
  outbox_event_days: 7
  import_job_days: 90
  trusted_device_days: 7
`))
	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "30 3 * * *", cfg.Cron)
	assert.Equal(t, 200, cfg.BatchSize)
	assert.Equal(t, 180, cfg.AuditLogDays)
	assert.Equal(t, 7, cfg.JobDays)
	assert.Equal(t, 30, cfg.JobHistoryDays)
	assert.Equal(t, 30, cfg.DeadLetterDays)
	assert.Equal(t, 7, cfg.OutboxEventDays)
	assert.Equal(t, 90, cfg.ImportJobDays)
	assert.Equal(t, 7, cfg.TrustedDeviceDays)
}

func TestLoadMissingSectionIsDisabled(t *testing.T) {
	cfg, err := Load(yamlSection(t, ""))
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
}
