package audit

import (
	"testing"

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
	assert.Equal(t, "audit", ConfigKey, "对外配置键不得变化")
}

func TestLoadValidAuditConfig(t *testing.T) {
	dec := &fakeDecoder{values: map[string]any{"audit": Config{QueueSize: 256, BatchSize: 50, FlushIntervalMS: 500, HashSecret: "s"}}}
	got, err := Load(dec)
	require.NoError(t, err)
	assert.Equal(t, 256, got.QueueSize)
	assert.Equal(t, "s", got.HashSecret)
}

// TestValidateAuditConfig 逐条覆盖原有校验语义（队列/批量/刷盘间隔）。
func TestValidateAuditConfig(t *testing.T) {
	valid := Config{QueueSize: 256, BatchSize: 50, FlushIntervalMS: 500}
	require.NoError(t, valid.Validate())

	for name, cfg := range map[string]Config{
		"queue_size 非正":     {QueueSize: 0, BatchSize: 1, FlushIntervalMS: 1},
		"batch_size 非正":     {QueueSize: 10, BatchSize: 0, FlushIntervalMS: 1},
		"batch_size 大于队列":   {QueueSize: 10, BatchSize: 11, FlushIntervalMS: 1},
		"flush_interval 非正": {QueueSize: 10, BatchSize: 1, FlushIntervalMS: 0},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, cfg.Validate())
			_, err := Load(&fakeDecoder{values: map[string]any{"audit": cfg}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "audit")
		})
	}
}

// TestApplyDefaultsReadsEnvOverride 段的环境覆盖随能力下沉（AUDIT_HASH_SECRET）。
func TestApplyDefaultsReadsEnvOverride(t *testing.T) {
	t.Setenv("AUDIT_HASH_SECRET", "from-env")
	got, err := Load(&fakeDecoder{values: map[string]any{"audit": Config{
		QueueSize: 256, BatchSize: 50, FlushIntervalMS: 500, HashSecret: "from-yaml",
	}}})
	require.NoError(t, err)
	assert.Equal(t, "from-env", got.HashSecret, "环境变量优先于 YAML")
}
