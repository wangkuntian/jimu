package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupTestCache(t *testing.T) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15, // 使用 DB 15 避免污染其他数据
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// 清理测试数据
	client.FlushDB(ctx)

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

	cache.Set(ctx, "key", "value", time.Minute)

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

func TestRedisCache_DeletePattern(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	cache.Set(ctx, "user:1", "a", time.Minute)
	cache.Set(ctx, "user:2", "b", time.Minute)
	cache.Set(ctx, "post:1", "c", time.Minute)

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
