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
	Enabled   bool
	Cron      string
	BatchSize int
}

// TestLoadSectionAppliesDefaultsBeforeValidate 默认值在校验前生效。
func TestLoadSectionAppliesDefaultsBeforeValidate(t *testing.T) {
	dec := &fakeSectionDecoder{values: map[string]any{"retention": retentionSection{Enabled: true}}}
	var got retentionSection
	validated := false

	err := LoadSection(dec, "retention", &got, func() {
		if got.BatchSize == 0 {
			got.BatchSize = 500
		}
	}, func() error {
		validated = true
		require.Equal(t, 500, got.BatchSize, "校验时必须已应用默认值")
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "retention", dec.seen)
	assert.Equal(t, 500, got.BatchSize)
	assert.True(t, validated)
}

// TestLoadSectionDottedKey 点分键（嵌套段归不同能力）可用。
func TestLoadSectionDottedKey(t *testing.T) {
	dec := &fakeSectionDecoder{}
	var got retentionSection
	require.NoError(t, LoadSection(dec, "auth.webauthn", &got, nil, nil))
	assert.Equal(t, "auth.webauthn", dec.seen)
}

// TestLoadSectionErrors 解码与校验错误都上抛，且带段名便于定位。
func TestLoadSectionErrors(t *testing.T) {
	var got retentionSection
	err := LoadSection(&fakeSectionDecoder{err: errors.New("boom")}, "retention", &got, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention")
	assert.Contains(t, err.Error(), "boom")

	err = LoadSection(&fakeSectionDecoder{}, "retention", &got, nil, func() error {
		return errors.New("bad cron")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention")
	assert.Contains(t, err.Error(), "bad cron")
}

// TestLoadSectionMissingIsZeroValue 段缺失时保持零值（=能力默认行为）。
func TestLoadSectionMissingIsZeroValue(t *testing.T) {
	var got retentionSection
	require.NoError(t, LoadSection(&fakeSectionDecoder{}, "scheduler", &got, func() {
		if got.Cron == "" {
			got.Cron = "0 * * * *"
		}
	}, nil))
	assert.Equal(t, "0 * * * *", got.Cron)
}
