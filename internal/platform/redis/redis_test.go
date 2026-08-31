// internal/platform/redis/redis_test.go
package redis

import (
	"context"
	"testing"
	"time"

	"jimu/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

func TestNew_SingleMode(t *testing.T) {
	cfg := config.RedisConfig{
		Mode:            config.RedisModeSingle,
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

	// 单机模式应返回 go-redis 普通客户端，可通过类型断言检查
	client, ok := c.(*redis.Client)
	require.True(t, ok, "single mode should produce *redis.Client")
	opt := client.Options()
	assert.Equal(t, cfg.Addr, opt.Addr)
	assert.Equal(t, cfg.Password, opt.Password)
	assert.Equal(t, cfg.DB, opt.DB)
	assert.Equal(t, cfg.PoolSize, opt.PoolSize)
	assert.Equal(t, cfg.MinIdleConns, opt.MinIdleConns)
	assert.Equal(t, 5*time.Second, opt.ReadTimeout)
	assert.Equal(t, 7*time.Second, opt.WriteTimeout)
}

func TestNew_DefaultModeIsSingle(t *testing.T) {
	// Mode 为空时应回退为单机模式
	c := New(config.RedisConfig{Addr: "127.0.0.1:6379"})
	defer c.Close()
	_, ok := c.(*redis.Client)
	assert.True(t, ok, "empty mode should default to single client")
}

func TestNew_SentinelMode(t *testing.T) {
	cfg := config.RedisConfig{
		Mode:            config.RedisModeSentinel,
		MasterName:      "mymaster",
		SentinelAddrs:   []string{"127.0.0.1:26379"},
		Password:        "secret",
		DB:              1,
		PoolSize:        8,
		ReadTimeoutSec:  4,
		WriteTimeoutSec: 6,
	}
	c := New(cfg)
	defer c.Close()

	client, ok := c.(*redis.Client)
	require.True(t, ok, "sentinel mode should produce failover client (typed *redis.Client)")
	opt := client.Options()
	// go-redis 哨兵模式下 Options.Addr 是 "FailoverClient" 占位符，DB 等常规选项仍保留
	assert.Equal(t, "FailoverClient", opt.Addr)
	assert.Equal(t, cfg.DB, opt.DB)
}

func TestNew_ClusterMode(t *testing.T) {
	cfg := config.RedisConfig{
		Mode:            config.RedisModeCluster,
		ClusterAddrs:    []string{"127.0.0.1:7000", "127.0.0.1:7001", "127.0.0.1:7002"},
		Password:        "secret",
		PoolSize:        8,
		ReadTimeoutSec:  4,
		WriteTimeoutSec: 6,
	}
	c := New(cfg)
	defer c.Close()

	// 集群模式应返回 go-redis 集群客户端
	_, ok := c.(*redis.ClusterClient)
	assert.True(t, ok, "cluster mode should produce *redis.ClusterClient")
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
