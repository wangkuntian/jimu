// internal/platform/redis/lock_test.go
package redis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLock(t *testing.T) (*Lock, *redis.Client) {
	t.Helper()
	mr := newTestClient(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewLock(client, "testlock"), client
}

func TestLock_AcquireRelease(t *testing.T) {
	l, _ := newLock(t)
	ctx := context.Background()

	res, err := l.TryAcquire(ctx, "job:1", time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Key)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "testlock:job:1", res.Key)

	// 未释放时再次获取应失败
	_, err = l.TryAcquire(ctx, "job:1", time.Second)
	require.Error(t, err)

	// 释放后可重新获取
	require.NoError(t, l.Release(ctx, res))
	res2, err := l.TryAcquire(ctx, "job:1", time.Second)
	require.NoError(t, err)
	assert.NotEqual(t, res.Token, res2.Token)
}

func TestLock_ReleaseOnlyOwnToken(t *testing.T) {
	l, client := newLock(t)
	ctx := context.Background()

	res, err := l.TryAcquire(ctx, "job:2", time.Second)
	require.NoError(t, err)

	// 用错误 token 直接改值后，Release 不应删除（Lua 校验 token）
	require.NoError(t, client.Set(ctx, res.Key, "someone-else", time.Minute).Err())
	require.NoError(t, l.Release(ctx, res))

	// 锁仍存在
	v, err := client.Get(ctx, res.Key).Result()
	require.NoError(t, err)
	assert.Equal(t, "someone-else", v)
}

func TestLock_Extend(t *testing.T) {
	l, client := newLock(t)
	ctx := context.Background()

	res, err := l.TryAcquire(ctx, "job:3", time.Second)
	require.NoError(t, err)

	require.NoError(t, l.Extend(ctx, res, 5*time.Second))
	ttl, err := client.TTL(ctx, res.Key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 3*time.Second)

	// token 不匹配时 Extend 报错
	require.NoError(t, client.Set(ctx, res.Key, "other", time.Minute).Err())
	err = l.Extend(ctx, res, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token mismatch")
}

func TestLock_WithLock(t *testing.T) {
	l, _ := newLock(t)
	ctx := context.Background()

	ran := false
	err := l.WithLock(ctx, "job:4", time.Second, func() error {
		ran = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, ran)

	// 锁已释放，可再次执行
	err = l.WithLock(ctx, "job:4", time.Second, func() error { return nil })
	require.NoError(t, err)

	// 业务错误透传，且锁仍被释放
	sentinel := errors.New("boom")
	err = l.WithLock(ctx, "job:4", time.Second, func() error { return sentinel })
	assert.ErrorIs(t, err, sentinel)
	_, err = l.TryAcquire(ctx, "job:4", time.Second)
	require.NoError(t, err)
}

func TestLock_ConcurrentAcquire(t *testing.T) {
	l, _ := newLock(t)
	ctx := context.Background()

	// 并发抢同一把锁，只应有一个成功
	acquired := make(chan bool, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := l.TryAcquire(ctx, "job:5", time.Second)
			acquired <- err == nil
		}()
	}
	wg.Wait()
	close(acquired)

	success := 0
	for ok := range acquired {
		if ok {
			success++
		}
	}
	assert.Equal(t, 1, success)
}
