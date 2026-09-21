package captcha

import (
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDecoder 返回预设段值。
type fakeDecoder struct {
	values map[string]any
	err    error
	seen   string
}

func (d *fakeDecoder) UnmarshalKey(key string, rawVal any) error {
	d.seen = key
	if d.err != nil {
		return d.err
	}
	if p, ok := rawVal.(*Config); ok {
		if v, ok := d.values[key].(Config); ok {
			*p = v
		}
	}
	return nil
}

func TestConfigKeyIsStable(t *testing.T) {
	// 对外配置键不得变化（设计 §11）
	assert.Equal(t, "captcha", ConfigKey)
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

func TestLoadDecodesEnabledCaptcha(t *testing.T) {
	dec := &fakeDecoder{values: map[string]any{"captcha": Config{Enabled: true, TTLMin: 5}}}
	got, err := loadConfig(dec)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, 5, got.TTLMin)
	assert.Equal(t, "captcha", dec.seen)
}

// TestValidateRejectsZeroTTLWhenEnabled 启用时 ttl_min 必须为正。
func TestValidateRejectsZeroTTLWhenEnabled(t *testing.T) {
	err := Config{Enabled: true, TTLMin: 0}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "captcha.ttl_min")

	_, err = loadConfig(&fakeDecoder{values: map[string]any{"captcha": Config{Enabled: true}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "captcha")
}

// TestValidateAllowsZeroTTLWhenDisabled 未启用时不校验（原语义：仅 enabled 时校验）。
func TestValidateAllowsZeroTTLWhenDisabled(t *testing.T) {
	require.NoError(t, Config{Enabled: false, TTLMin: 0}.Validate())

	got, err := loadConfig(&fakeDecoder{values: map[string]any{"captcha": Config{Enabled: false}}})
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

// TestLoadMissingSectionIsZeroValue 段缺失时零值（Enabled=false，不启用）。
func TestLoadMissingSectionIsZeroValue(t *testing.T) {
	got, err := loadConfig(&fakeDecoder{})
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	assert.Zero(t, got.TTLMin)
}
