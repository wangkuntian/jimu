package outbox

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
	assert.Equal(t, "outbox", ConfigKey, "对外配置键不得变化")
}

// TestValidateOutboxPublisher 迁移自 internal/config 的同名用例。
func TestValidateOutboxPublisher(t *testing.T) {
	require.NoError(t, Config{Publisher: PublisherEventBus}.Validate())
	require.NoError(t, Config{Publisher: PublisherMQ}.Validate())

	_, err := Load(&fakeDecoder{values: map[string]any{"outbox": Config{Publisher: "invalid"}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outbox.publisher")
}

func TestUsesMQ(t *testing.T) {
	assert.True(t, Config{Publisher: PublisherMQ}.UsesMQ())
	assert.False(t, Config{Publisher: PublisherEventBus}.UsesMQ())
}

func TestLoadMissingSectionRejected(t *testing.T) {
	// publisher 必填：段缺失时零值非法（与下沉前一致）
	_, err := Load(&fakeDecoder{})
	require.Error(t, err)
}
