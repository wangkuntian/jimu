package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestCache(t *testing.T) *RedisCache {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run failed: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewRedisCache(client, "test")
}

func TestRedisCache_SetAndGet(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	type User struct {
		ID   uint64 `json:"id"`
		Name string `json:"name"`
	}

	user := User{ID: 1, Name: "test"}
	err := cache.Set(ctx, "user:1", user, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var result User
	found, err := cache.Get(ctx, "user:1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if result.ID != user.ID || result.Name != user.Name {
		t.Errorf("got %+v, want %+v", result, user)
	}
}

func TestRedisCache_Delete(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err := cache.Delete(ctx, "key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, _ := cache.Exists(ctx, "key")
	if exists {
		t.Error("expected key to be deleted")
	}
}

func TestRedisCache_GetOrSet(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	type Config struct {
		Value string `json:"value"`
	}

	callCount := 0
	fetch := func() (interface{}, error) {
		callCount++
		return Config{Value: "fetched"}, nil
	}

	var config Config
	err := cache.GetOrSet(ctx, "config", &config, time.Minute, fetch)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected fetch to be called once, got %d", callCount)
	}

	// 第二次应该从缓存获取
	err = cache.GetOrSet(ctx, "config", &config, time.Minute, fetch)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected fetch to still be called once (from cache), got %d", callCount)
	}
}

func TestRedisCache_GetOrSetStampede(t *testing.T) {
	// 并发 GetOrSet 防击穿：锁保证只有一个请求执行 fetch
	cache := setupTestCache(t)
	ctx := context.Background()

	type Config struct {
		Value string `json:"value"`
	}

	var mu sync.Mutex
	callCount := 0
	fetch := func() (interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(30 * time.Millisecond) // 模拟慢回源
		return Config{Value: "fetched"}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var cfg Config
			err := cache.GetOrSet(ctx, "config", &cfg, time.Minute, fetch)
			assert.NoError(t, err)
			assert.Equal(t, "fetched", cfg.Value)
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, callCount, "并发防击穿应只回源一次")
}

func TestRedisCache_DeletePattern(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	if err := cache.Set(ctx, "user:1", "a", time.Minute); err != nil {
		t.Fatalf("Set user:1 failed: %v", err)
	}
	if err := cache.Set(ctx, "user:2", "b", time.Minute); err != nil {
		t.Fatalf("Set user:2 failed: %v", err)
	}
	if err := cache.Set(ctx, "post:1", "c", time.Minute); err != nil {
		t.Fatalf("Set post:1 failed: %v", err)
	}

	err := cache.DeletePattern(ctx, "user:*")
	if err != nil {
		t.Fatalf("DeletePattern failed: %v", err)
	}

	exists, _ := cache.Exists(ctx, "user:1")
	if exists {
		t.Error("expected user:1 to be deleted")
	}
	exists, _ = cache.Exists(ctx, "post:1")
	if !exists {
		t.Error("expected post:1 to still exist")
	}
}
