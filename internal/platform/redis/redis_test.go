// internal/platform/redis/redis_test.go
package redis

import (
	"context"
	"testing"
	"time"

	"jimu/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient 创建基于 miniredis 的测试客户端（无需外部 Redis）
func newTestClient(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run failed: %v", err)
	}
	t.Cleanup(mr.Close)
	return mr
}

func TestNew_SetsOptions(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:            "127.0.0.1:6379",
		Password:        "secret",
		DB:              3,
		PoolSize:        10,
		MinIdleConns:    2,
		ReadTimeoutSec:  5,
		WriteTimeoutSec: 7,
	}
	c := New(cfg)
	defer c.Close()

	opt := c.Options()
	assert.Equal(t, cfg.Addr, opt.Addr)
	assert.Equal(t, cfg.Password, opt.Password)
	assert.Equal(t, cfg.DB, opt.DB)
	assert.Equal(t, cfg.PoolSize, opt.PoolSize)
	assert.Equal(t, cfg.MinIdleConns, opt.MinIdleConns)
	assert.Equal(t, 5*time.Second, opt.ReadTimeout)
	assert.Equal(t, 7*time.Second, opt.WriteTimeout)
}

func TestConnectWithRetry_Success(t *testing.T) {
	mr := newTestClient(t)

	client, err := ConnectWithRetry(config.RedisConfig{Addr: mr.Addr(), MaxRetries: 1}, nil)
	require.NoError(t, err)
	defer client.Close()

	assert.NoError(t, client.Ping(context.Background()).Err())
}

func TestConnectWithRetry_Exhausted(t *testing.T) {
	// 端口 1 通常不可达，快速返回 connection refused
	_, err := ConnectWithRetry(config.RedisConfig{Addr: "127.0.0.1:1", MaxRetries: 1, RetryIntervalSec: 1}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis connection failed after 1 attempts")
}
