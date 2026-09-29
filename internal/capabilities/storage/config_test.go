package storage

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// viperSection 把 viper 适配为 config.SectionDecoder（viper 的 UnmarshalKey
// 带变参，不直接满足接口；production 侧由 config.LoadWithSections 包装）。
type viperSection struct{ v *viper.Viper }

func (s viperSection) UnmarshalKey(key string, rawVal any) error {
	return s.v.UnmarshalKey(key, rawVal)
}

// yamlSectionDecoder 用真实 viper 按段解码，验证 YAML 键名映射（随段下沉自
// internal/config 的 TestStorageConfigFieldMapping，键名必须保持不变）。
func yamlSectionDecoder(t *testing.T, yaml string) viperSection {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(yaml)))
	return viperSection{v: v}
}

func TestConfigKeyIsStable(t *testing.T) {
	assert.Equal(t, "storage", ConfigKey, "对外配置键不得变化")
}

func TestLoadMapsS3Fields(t *testing.T) {
	v := yamlSectionDecoder(t, `
storage:
  type: "oss"
  endpoint: "oss-cn-hangzhou.aliyuncs.com"
  region: "cn-hangzhou"
  bucket: "my-bucket"
  access_key: "ak"
  secret_key: "sk"
  path_style: true
`)
	cfg, err := Load(v)
	require.NoError(t, err)
	assert.Equal(t, StorageType("oss"), cfg.Type)
	assert.Equal(t, "oss-cn-hangzhou.aliyuncs.com", cfg.Endpoint)
	assert.Equal(t, "my-bucket", cfg.Bucket)
	assert.Equal(t, "cn-hangzhou", cfg.Region)
	assert.True(t, cfg.PathStyle)
	assert.Equal(t, "ak", cfg.AccessKey)
	assert.Equal(t, "sk", cfg.SecretKey)
}

func TestLoadMapsLocalFields(t *testing.T) {
	v := yamlSectionDecoder(t, `
storage:
  type: "local"
  base_dir: "/data/files"
  base_url: "https://cdn.example.com"
`)
	cfg, err := Load(v)
	require.NoError(t, err)
	assert.Equal(t, StorageTypeLocal, cfg.Type)
	assert.Equal(t, "/data/files", cfg.BaseDir)
	assert.Equal(t, "https://cdn.example.com", cfg.BaseURL)
}

// TestLoadDoesNotValidate 存储驱动合法性由 New 判定，Load 不额外校验。
func TestLoadDoesNotValidate(t *testing.T) {
	cfg, err := Load(yamlSectionDecoder(t, "storage:\n  type: \"bogus\"\n"))
	require.NoError(t, err)
	_, err = New(*cfg)
	assert.Error(t, err, "未知驱动应由 New 拒绝")
}
