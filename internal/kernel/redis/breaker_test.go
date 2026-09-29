package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"jimu/internal/kernel/breaker"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisBreakerOpensOnTransportFailure(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		MaxRetries:   -1, // 关闭内部重试，让每次命令只上报一次失败
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	b := AttachBreaker(client, breaker.Config{MaxFailures: 2, ResetTimeout: time.Minute})
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, "k", "v", time.Minute).Err())
	assert.Equal(t, breaker.Closed, b.State())

	// 关闭 Redis：后续命令为连接类错误
	mr.Close()

	_ = client.Get(ctx, "k").Err()
	_ = client.Get(ctx, "k").Err()
	assert.Equal(t, breaker.Open, b.State(), "连续传输失败应触发熔断")

	assert.ErrorIs(t, client.Get(ctx, "k").Err(), breaker.ErrOpen, "熔断开启后应快速失败")
}

func TestRedisBreakerIgnoresMissAndBusinessError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	b := AttachBreaker(client, breaker.Config{MaxFailures: 2, ResetTimeout: time.Minute})
	ctx := context.Background()

	// 未命中（redis.Nil）不应计为失败
	for i := 0; i < 5; i++ {
		assert.ErrorIs(t, client.Get(ctx, "missing").Err(), redis.Nil)
	}
	assert.Equal(t, breaker.Closed, b.State())

	// 类型错误（业务错误）也不应计为失败
	require.NoError(t, client.Set(ctx, "str", "v", time.Minute).Err())
	for i := 0; i < 5; i++ {
		_ = client.LPop(ctx, "str").Err()
	}
	assert.Equal(t, breaker.Closed, b.State())
}

func TestIsTransportError(t *testing.T) {
	assert.False(t, IsTransportError(nil))
	assert.False(t, IsTransportError(redis.Nil))
	assert.False(t, IsTransportError(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")))
	assert.True(t, IsTransportError(redis.ErrClosed))
	assert.True(t, IsTransportError(errors.New("dial tcp 127.0.0.1:6379: connect: connection refused")))
	assert.True(t, IsTransportError(context.DeadlineExceeded))
}
