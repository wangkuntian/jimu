package application

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newConfigTestService(t *testing.T) (*AdminConfigService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewAdminConfigService(client, &fakeEventBus{}, "jimu"), mr
}

func TestAdminConfigServiceGetAllConfig(t *testing.T) {
	ctx := context.Background()
	svc, mr := newConfigTestService(t)

	// 空 Redis 返回空 map
	all, err := svc.GetAllConfig(ctx)
	assert.NoError(t, err)
	assert.Empty(t, all)

	// 有配置 key（shortKey 剥掉 "jimu:config:" 前缀）
	assert.NoError(t, mr.Set("jimu:config:rate_limit_rate", "100"))
	assert.NoError(t, mr.Set("jimu:config:unrelated", "x"))
	assert.NoError(t, mr.Set("other:config:rate_limit_rate", "999"))
	all, err = svc.GetAllConfig(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "100", all["rate_limit_rate"])
	assert.Equal(t, "x", all["unrelated"])
	// 非 jimu 前缀的 key 被跳过
	_, ok := all["other:config:rate_limit_rate"]
	assert.False(t, ok)
}

func TestAdminConfigServiceUpdateConfig(t *testing.T) {
	ctx := context.Background()
	svc, _ := newConfigTestService(t)
	eb := &fakeEventBus{}
	svc.eventBus = eb

	err := svc.UpdateConfig(ctx, "log_level", "debug")
	assert.NoError(t, err)
	assert.Equal(t, "debug", svc.redis.Get(ctx, "jimu:config:log_level").Val())
	assert.Equal(t, []string{"config.updated"}, eb.published())
}

func TestAdminConfigServiceReloadConfig(t *testing.T) {
	ctx := context.Background()
	svc, mr := newConfigTestService(t)
	eb := &fakeEventBus{}
	svc.eventBus = eb

	assert.NoError(t, mr.Set("jimu:config:log_level", "info"))
	assert.NoError(t, svc.ReloadConfig(ctx))
	assert.Equal(t, []string{"config.updated"}, eb.published())

	// Redis 不可用时报错
	svc = NewAdminConfigService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &fakeEventBus{}, "jimu")
	mr.Close()
	assert.Error(t, svc.ReloadConfig(ctx))
}

func TestAdminConfigServiceUpdateConfigError(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	svc := NewAdminConfigService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &fakeEventBus{}, "jimu")
	mr.Close()
	assert.Error(t, svc.UpdateConfig(ctx, "log_level", "debug"))
}

func TestAdminConfigServiceIsValidKey(t *testing.T) {
	svc := &AdminConfigService{}
	for _, key := range []string{"rate_limit_rate", "rate_limit_burst", "log_level", "feature_flags"} {
		assert.True(t, svc.IsValidKey(key))
	}
	assert.False(t, svc.IsValidKey("nope"))
	assert.False(t, svc.IsValidKey(""))
}

func TestAdminConfigServiceConfigKey(t *testing.T) {
	svc := NewAdminConfigService(nil, nil, "jimu")
	assert.Equal(t, "jimu:config:log_level", svc.configKey("log_level"))
}
