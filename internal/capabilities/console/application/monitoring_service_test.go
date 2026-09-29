package application

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestAdminMonitoringServiceGetStatus(t *testing.T) {
	svc := NewAdminMonitoringService("v1.2.3", "test", nil)
	status := svc.GetStatus()

	assert.Equal(t, "v1.2.3", status.Version)
	assert.Equal(t, "test", status.Environment)
	assert.False(t, status.StartTime.IsZero())
	assert.NotEmpty(t, status.Uptime)
	assert.Greater(t, status.NumCPU, 0)
	assert.Greater(t, status.NumGoroutine, 0)
	assert.False(t, status.DBConnected)
	assert.False(t, status.RedisConnected)
	assert.Greater(t, status.Memory.HeapAlloc, uint64(0))
}

func TestAdminMonitoringServiceGetHealth(t *testing.T) {
	ctx := context.Background()

	// 无 Redis 依赖
	svc := NewAdminMonitoringService("v1", "test", nil)
	health := svc.GetHealth(ctx)
	assert.False(t, health.Redis)

	// Redis 可用
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	svc = NewAdminMonitoringService("v1", "test", client)
	health = svc.GetHealth(ctx)
	assert.True(t, health.Redis)

	// Redis 不可用
	badClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close()
	svc = NewAdminMonitoringService("v1", "test", badClient)
	health = svc.GetHealth(ctx)
	assert.False(t, health.Redis)
}
