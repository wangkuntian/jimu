package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsUnregisteredDriver(t *testing.T) {
	assert.Empty(t, RegisteredTypes(), "核心包不得自注册任何驱动")
	_, err := New(Config{Type: TypeKafka})
	require.ErrorContains(t, err, `queue driver "kafka" is not compiled into this build`)
	require.ErrorContains(t, err, "compiled: none")
	assert.ErrorIs(t, EnsureRegistered(TypeRedis), ErrUnregisteredDriver) // 未注册即标记该哨兵错误
}

func TestNewUsesRegisteredDriver(t *testing.T) {
	Register(TypeRedis, func(Config) (Queue, error) { return nil, nil })
	t.Cleanup(func() { delete(drivers, TypeRedis) })
	q, err := New(Config{Type: TypeRedis})
	require.NoError(t, err)
	assert.Nil(t, q)
	assert.NoError(t, EnsureRegistered(TypeRedis))
}

func TestRegisterPanicsOnDuplicate(t *testing.T) {
	f := func(Config) (Queue, error) { return nil, nil }
	Register("dup", f)
	t.Cleanup(func() { delete(drivers, "dup") })
	assert.Panics(t, func() { Register("dup", f) })
}
