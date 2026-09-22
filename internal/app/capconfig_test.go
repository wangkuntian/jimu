package app

import (
	"errors"
	"strings"
	"testing"

	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/captcha"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/config"
	"jimu/internal/contract"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSectionConfig 记录钩子调用顺序，可注入校验错误。
type fakeSectionConfig struct {
	value           string
	defaultsApplied bool
	validated       bool
	validateErr     error
	prodValidated   bool
	prodErr         error
}

func (c *fakeSectionConfig) ApplyDefaults() { c.defaultsApplied = true }
func (c *fakeSectionConfig) Validate() error {
	c.validated = true
	return c.validateErr
}
func (c *fakeSectionConfig) ValidateProd() error {
	c.prodValidated = true
	return c.prodErr
}

// fakeDecoder 按段键填充预设值。
type fakeDecoder struct {
	values map[string]string
	err    error
	seen   []string
}

func (d *fakeDecoder) UnmarshalKey(key string, rawVal any) error {
	d.seen = append(d.seen, key)
	if d.err != nil {
		return d.err
	}
	if v, ok := rawVal.(*fakeSectionConfig); ok {
		v.value = d.values[key]
	}
	return nil
}

func desc(name, section string, newFn func() any) contract.Descriptor {
	return contract.Descriptor{
		Name:    name,
		Configs: []contract.ConfigSpec{{Section: section, New: newFn}},
	}
}

// TestLoadCapabilityConfigsDecodesDefaultsValidates 顺序：解码 → 默认值 → 校验。
func TestLoadCapabilityConfigsDecodesDefaultsValidates(t *testing.T) {
	dec := &fakeDecoder{values: map[string]string{"retention": "from-yaml"}}
	caps := []contract.Descriptor{desc("retention", "retention", func() any { return &fakeSectionConfig{} })}

	got, err := LoadCapabilityConfigs(dec, caps, "dev")
	require.NoError(t, err)
	require.Equal(t, 1, got.Len())

	sc, ok := SectionOf[*fakeSectionConfig](got, "retention")
	require.True(t, ok)
	assert.Equal(t, "from-yaml", sc.value)
	assert.True(t, sc.defaultsApplied)
	assert.True(t, sc.validated)
	assert.False(t, sc.prodValidated, "dev 环境不跑 prod 加严校验")
	assert.Equal(t, []string{"retention"}, dec.seen)
}

// TestLoadCapabilityConfigsProdHook prod 环境调用可选加严校验。
func TestLoadCapabilityConfigsProdHook(t *testing.T) {
	caps := []contract.Descriptor{desc("auth", "auth", func() any { return &fakeSectionConfig{} })}

	got, err := LoadCapabilityConfigs(&fakeDecoder{}, caps, "prod")
	require.NoError(t, err)
	sc, _ := SectionOf[*fakeSectionConfig](got, "auth")
	assert.True(t, sc.prodValidated)

	// prod 加严校验失败则上抛
	dec := &fakeDecoder{}
	caps = []contract.Descriptor{desc("auth", "auth", func() any {
		return &fakeSectionConfig{prodErr: errors.New("weak secret")}
	})}
	_, err = LoadCapabilityConfigs(dec, caps, "prod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth")
	assert.Contains(t, err.Error(), "weak secret")
}

// TestLoadCapabilityConfigsSkipsDisabled 未启用（不在 caps 内）的能力段既不加载也不校验。
func TestLoadCapabilityConfigsSkipsDisabled(t *testing.T) {
	dec := &fakeDecoder{}
	// 仅启用 user（无配置段）；retention 未启用
	got, err := LoadCapabilityConfigs(dec, []contract.Descriptor{{Name: "user"}}, "dev")
	require.NoError(t, err)
	assert.Zero(t, got.Len())
	assert.Empty(t, dec.seen, "未启用能力的配置段不得被解码")
	assert.Nil(t, got.Section("retention"))
}

// TestLoadCapabilityConfigsMultipleSections 一个能力可拥有多段（queue + scheduler）。
func TestLoadCapabilityConfigsMultipleSections(t *testing.T) {
	caps := []contract.Descriptor{{
		Name: "queue",
		Configs: []contract.ConfigSpec{
			{Section: "queue", New: func() any { return &fakeSectionConfig{} }},
			{Section: "scheduler", New: func() any { return &fakeSectionConfig{} }},
		},
	}}
	got, err := LoadCapabilityConfigs(&fakeDecoder{}, caps, "dev")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Len())
	_, ok := SectionOf[*fakeSectionConfig](got, "scheduler")
	assert.True(t, ok)
}

// TestLoadCapabilityConfigsErrors 声明缺陷与校验失败都 fail fast。
func TestLoadCapabilityConfigsErrors(t *testing.T) {
	// 未实现 SectionConfig
	_, err := LoadCapabilityConfigs(&fakeDecoder{}, []contract.Descriptor{
		desc("bad", "bad", func() any { return &struct{}{} }),
	}, "dev")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SectionConfig")

	// 空段键
	_, err = LoadCapabilityConfigs(&fakeDecoder{}, []contract.Descriptor{
		{Name: "bad", Configs: []contract.ConfigSpec{{Section: "", New: func() any { return &fakeSectionConfig{} }}}},
	}, "dev")
	require.Error(t, err)

	// 校验失败带段名
	_, err = LoadCapabilityConfigs(&fakeDecoder{}, []contract.Descriptor{
		desc("captcha", "captcha", func() any { return &fakeSectionConfig{validateErr: errors.New("bad ttl")} }),
	}, "dev")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "captcha")
	assert.Contains(t, err.Error(), "bad ttl")

	// 解码失败
	_, err = LoadCapabilityConfigs(&fakeDecoder{err: errors.New("boom")}, []contract.Descriptor{
		desc("captcha", "captcha", func() any { return &fakeSectionConfig{} }),
	}, "dev")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

var _ config.SectionConfig = (*fakeSectionConfig)(nil)
var _ config.ProdConfigValidator = (*fakeSectionConfig)(nil)

// viperSectionDecoder 把 viper 适配为 config.SectionDecoder，用真实 YAML 解码能力段。
type viperSectionDecoder struct{ v *viper.Viper }

func (d viperSectionDecoder) UnmarshalKey(key string, rawVal any) error {
	return d.v.UnmarshalKey(key, rawVal)
}

func newYAMLDecoder(t *testing.T, document string) config.SectionDecoder {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(document)))
	return viperSectionDecoder{v: v}
}

// TestLoadCapabilityConfigsProdEnforcesAuthJWTSecret 生产路径首次使用 ProdConfigValidator：
// auth 段经 Descriptor.Configs 声明并走 LoadCapabilityConfigs，APP_ENV=prod 时
// jwt_secret 强度检查必须真正生效（不能只在 dev 跑绿）。
func TestLoadCapabilityConfigsProdEnforcesAuthJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")

	const doc = `
auth:
  jwt_secret: "short"
  issuer: "jimu"
  access_expire_min: 30
  refresh_expire_day: 7
  login_rate_limit: 10
  login_rate_window_sec: 60
  register_rate_limit: 5
  register_rate_window_sec: 300
  reset_code_ttl_min: 15
`
	caps := []contract.Descriptor{authmodule.Descriptor}

	// dev：弱密钥可启动（prod 专属加严不得泄漏到其它环境）
	if _, err := LoadCapabilityConfigs(newYAMLDecoder(t, doc), caps, "dev"); err != nil {
		t.Fatalf("dev 环境不应执行 prod 加严校验: %v", err)
	}

	// prod：弱密钥必须 fail-closed
	_, err := LoadCapabilityConfigs(newYAMLDecoder(t, doc), caps, "prod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.jwt_secret")

	// prod + 强密钥：通过，且段可被强类型取回
	strong := strings.Replace(doc, `jwt_secret: "short"`, `jwt_secret: "`+strings.Repeat("a", 32)+`"`, 1)
	got, err := LoadCapabilityConfigs(newYAMLDecoder(t, strong), caps, "prod")
	require.NoError(t, err)
	cfg, ok := SectionOf[*authmodule.Config](got, authmodule.ConfigKey)
	require.True(t, ok)
	assert.Equal(t, "jimu", cfg.Issuer)
}

// TestLoadCapabilityConfigsSkipsDisabledRealDescriptors 装配级回归（设计 §8）：
// 未启用能力的配置段既不出现也不校验 —— 即使 YAML 中有非法配置，只要对应能力未启用，
// 启动就不得失败；同一份配置在能力启用后必须 fail-closed。
func TestLoadCapabilityConfigsSkipsDisabledRealDescriptors(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")

	const doc = `
captcha:
  enabled: true
  ttl_min: 0        # 非法：captcha 启用时 ttl_min 必须为正
audit:
  queue_size: 10
  batch_size: 99    # 非法：batch_size > queue_size
`

	// 只启用 user：captcha/audit 未启用，非法段不得被解码或校验
	caps, err := catalog.Resolve([]string{"user"})
	require.NoError(t, err)
	got, err := LoadCapabilityConfigs(newYAMLDecoder(t, doc), caps, "dev")
	require.NoError(t, err)
	assert.Zero(t, got.Len(), "未启用能力的配置段不得出现")
	assert.Nil(t, got.Section(captcha.ConfigKey))
	assert.Nil(t, got.Section(auditmodule.ConfigKey))

	// 启用 captcha 后同一份非法配置必须 fail-closed
	caps, err = catalog.Resolve([]string{"user", "captcha"})
	require.NoError(t, err)
	_, err = LoadCapabilityConfigs(newYAMLDecoder(t, doc), caps, "dev")
	require.Error(t, err)
	assert.Contains(t, err.Error(), captcha.ConfigKey)
}
