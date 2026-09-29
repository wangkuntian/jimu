package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSectionDecoder 按预设键值返回解码结果，可注入解码错误。
type fakeSectionDecoder struct {
	values map[string]any
	err    error
	seen   string
}

func (d *fakeSectionDecoder) UnmarshalKey(key string, rawVal any) error {
	d.seen = key
	if d.err != nil {
		return d.err
	}
	switch p := rawVal.(type) {
	case *retentionSection:
		if v, ok := d.values[key].(retentionSection); ok {
			*p = v
		}
	}
	return nil
}

type retentionSection struct {
	Enabled         bool
	Cron            string
	BatchSize       int
	defaultsApplied bool
	validated       bool
}

func (c *retentionSection) ApplyDefaults() {
	c.defaultsApplied = true
	if c.BatchSize == 0 {
		c.BatchSize = 500
	}
}

func (c *retentionSection) Validate() error {
	c.validated = true
	if c.Enabled && c.Cron == "" {
		return errors.New("cron required")
	}
	return nil
}

// TestLoadSectionAppliesDefaultsBeforeValidate 默认值在校验前生效，
// 且钩子作用于解码后的实际值（回归：方法值会绑定零值接收者副本）。
func TestLoadSectionAppliesDefaultsBeforeValidate(t *testing.T) {
	dec := &fakeSectionDecoder{values: map[string]any{"retention": retentionSection{Enabled: true, Cron: "30 3 * * *"}}}
	var got retentionSection

	require.NoError(t, LoadSection(dec, "retention", &got))
	assert.Equal(t, "retention", dec.seen)
	assert.Equal(t, 500, got.BatchSize, "默认值应已应用")
	assert.True(t, got.defaultsApplied)
	assert.True(t, got.validated)
}

// TestLoadSectionValidateSeesDecodedValue 校验必须看到解码值（非零值副本）。
func TestLoadSectionValidateSeesDecodedValue(t *testing.T) {
	dec := &fakeSectionDecoder{values: map[string]any{"retention": retentionSection{Enabled: true, Cron: ""}}}
	var got retentionSection
	err := LoadSection(dec, "retention", &got)
	require.Error(t, err, "启用且缺 cron 时必须校验失败")
	assert.Contains(t, err.Error(), "retention")
	assert.Contains(t, err.Error(), "cron required")
}

// TestLoadSectionDottedKey 点分键（嵌套段归不同能力）可用。
func TestLoadSectionDottedKey(t *testing.T) {
	dec := &fakeSectionDecoder{}
	var got retentionSection
	require.NoError(t, LoadSection(dec, "auth.webauthn", &got))
	assert.Equal(t, "auth.webauthn", dec.seen)
}

// TestLoadSectionErrors 解码与校验错误都上抛，且带段名便于定位。
func TestLoadSectionErrors(t *testing.T) {
	var got retentionSection
	err := LoadSection(&fakeSectionDecoder{err: errors.New("boom")}, "retention", &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention")
	assert.Contains(t, err.Error(), "boom")

	err = LoadSection(&fakeSectionDecoder{values: map[string]any{"retention": retentionSection{Enabled: true}}}, "retention", &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention")
	assert.Contains(t, err.Error(), "cron required")
}

// TestLoadSectionMissingIsZeroValue 段缺失时保持零值（=能力默认行为）。
func TestLoadSectionMissingIsZeroValue(t *testing.T) {
	var got retentionSection
	require.NoError(t, LoadSection(&fakeSectionDecoder{}, "scheduler", &got))
	assert.False(t, got.Enabled)
	assert.Equal(t, 500, got.BatchSize, "默认值仍应应用")
}
