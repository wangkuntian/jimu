package breaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	b := New("test", Config{MaxFailures: 3, ResetTimeout: time.Minute})

	for i := 0; i < 2; i++ {
		require.NoError(t, b.Allow())
		b.OnFailure()
	}
	assert.Equal(t, Closed, b.State())

	require.NoError(t, b.Allow())
	b.OnFailure()
	assert.Equal(t, Open, b.State())

	// 开启后快速失败
	assert.ErrorIs(t, b.Allow(), ErrOpen)
}

func TestBreakerSuccessResetsFailureCount(t *testing.T) {
	b := New("test", Config{MaxFailures: 3, ResetTimeout: time.Minute})

	require.NoError(t, b.Allow())
	b.OnFailure()
	require.NoError(t, b.Allow())
	b.OnFailure()
	require.NoError(t, b.Allow())
	b.OnSuccess() // 成功清零

	require.NoError(t, b.Allow())
	b.OnFailure()
	require.NoError(t, b.Allow())
	b.OnFailure()
	assert.Equal(t, Closed, b.State(), "成功应清零连续失败计数")
}

func TestBreakerHalfOpenProbeRecovers(t *testing.T) {
	b := New("test", Config{MaxFailures: 1, ResetTimeout: 20 * time.Millisecond})

	require.NoError(t, b.Allow())
	b.OnFailure()
	assert.ErrorIs(t, b.Allow(), ErrOpen)

	time.Sleep(30 * time.Millisecond)
	// 冷却结束：首个调用获得探测机会，其余拒绝
	require.NoError(t, b.Allow())
	assert.ErrorIs(t, b.Allow(), ErrOpen, "半开态只放行一个探测")
	b.OnSuccess()
	assert.Equal(t, Closed, b.State())
	require.NoError(t, b.Allow())
}

func TestBreakerHalfOpenProbeFailureReopens(t *testing.T) {
	b := New("test", Config{MaxFailures: 1, ResetTimeout: 20 * time.Millisecond})

	require.NoError(t, b.Allow())
	b.OnFailure()

	time.Sleep(30 * time.Millisecond)
	require.NoError(t, b.Allow())
	b.OnFailure()
	assert.Equal(t, Open, b.State())
	assert.ErrorIs(t, b.Allow(), ErrOpen)
}

func TestBreakerDefaults(t *testing.T) {
	b := New("test", Config{})
	assert.Equal(t, defaultMaxFailures, b.maxFailures)
	assert.Equal(t, defaultResetTimeout, b.resetTimeout)
	assert.Equal(t, Closed, b.State())
	assert.False(t, errors.Is(nil, ErrOpen))
}
