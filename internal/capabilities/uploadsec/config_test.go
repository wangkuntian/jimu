package uploadsec

import (
	"strings"
	"testing"
	"time"

	"jimu/internal/config"

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
	assert.Equal(t, "upload", ConfigKey, "对外配置键不得变化")
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

// TestLoadMapsClamAVSection YAML 键名与下沉前一致（enabled/address/timeout_sec）。
func TestLoadMapsClamAVSection(t *testing.T) {
	cfg, err := loadConfig(yamlSection(t, `
upload:
  clamav:
    enabled: true
    address: "clamd:3310"
    timeout_sec: 25
`))
	require.NoError(t, err)
	assert.True(t, cfg.ClamAV.Enabled)
	assert.Equal(t, "clamd:3310", cfg.ClamAV.Address)
	assert.Equal(t, 25, cfg.ClamAV.TimeoutSec)
}

// TestScannerNilWhenDisabled 未启用扫描时返回 nil（上传不扫描，向后兼容）。
func TestScannerNilWhenDisabled(t *testing.T) {
	cfg, err := loadConfig(yamlSection(t, "upload:\n  clamav:\n    enabled: false\n"))
	require.NoError(t, err)
	assert.Nil(t, cfg.Scanner())
}

// TestScannerConvertsSecondsToDuration 秒级超时换算为 Duration。
func TestScannerConvertsSecondsToDuration(t *testing.T) {
	cfg, err := loadConfig(yamlSection(t, `
upload:
  clamav:
    enabled: true
    address: "clamd:3310"
    timeout_sec: 25
`))
	require.NoError(t, err)
	scanner, ok := cfg.Scanner().(*ClamAVScanner)
	require.True(t, ok)
	assert.Equal(t, "clamd:3310", scanner.address)
	assert.Equal(t, 25*time.Second, scanner.timeout)
}

// TestLoadMissingSectionIsZeroValue 段缺失时零值（不扫描）。
func TestLoadMissingSectionIsZeroValue(t *testing.T) {
	cfg, err := loadConfig(yamlSection(t, ""))
	require.NoError(t, err)
	assert.False(t, cfg.ClamAV.Enabled)
	assert.Nil(t, cfg.Scanner())
}
