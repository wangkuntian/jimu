package app

import (
	"errors"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"

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
