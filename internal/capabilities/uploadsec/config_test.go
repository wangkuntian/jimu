package uploadsec

import (
	"strings"
	"testing"
	"time"

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

// TestLoadMapsClamAVSection YAML 键名与下沉前一致（enabled/address/timeout_sec）。
func TestLoadMapsClamAVSection(t *testing.T) {
	cfg, err := Load(yamlSection(t, `
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
	cfg, err := Load(yamlSection(t, "upload:\n  clamav:\n    enabled: false\n"))
	require.NoError(t, err)
	assert.Nil(t, cfg.Scanner())
}

// TestScannerConvertsSecondsToDuration 秒级超时换算为 Duration。
func TestScannerConvertsSecondsToDuration(t *testing.T) {
	cfg, err := Load(yamlSection(t, `
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
	cfg, err := Load(yamlSection(t, ""))
	require.NoError(t, err)
	assert.False(t, cfg.ClamAV.Enabled)
	assert.Nil(t, cfg.Scanner())
}
