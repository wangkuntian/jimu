package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// 防击穿参数
const (
	lockTTL      = 5 * time.Second // 锁有效期，防止持锁者崩溃导致死锁
	lockRetry    = 3               // 未拿到锁的最大等待轮次
	lockWaitTime = 100 * time.Millisecond
	lockPrefix   = "lock:" // 锁 key 前缀，与业务缓存 key 隔离
)

// releaseLockScript 原子释放锁：仅当锁 token 匹配时删除，防止误删他人锁
var releaseLockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
end
return 0
`)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error
	Close() error
}

// RedisCache 基于 Redis 的缓存实现
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// key 生成带前缀的 key
func (c *RedisCache) key(k string) string {
	return fmt.Sprintf("%s:%s", c.prefix, k)
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := c.client.Get(ctx, c.key(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(val), dest)
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

// DeletePattern 按模式删除缓存
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, c.key(pattern), 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists 检查 key 是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

// GetOrSet 缓存不存在时自动获取并设置（Cache-Aside 模式）。
// 防击穿：并发未命中时，只有一个请求执行 fetch，其余请求短暂等待后重读缓存；
// 等待超时则直接 fetch 兜底，不阻塞调用方。锁带 TTL 防死锁。
func (c *RedisCache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error {
	for {
		found, err := c.Get(ctx, key, dest)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		token, ok, err := c.acquireLock(ctx, key)
		if err != nil {
			// 锁服务异常时直接 fetch，保证可用性
			return c.fetchAndSet(ctx, key, dest, ttl, fetch)
		}
		if ok {
			// 拿到锁，成为回源者
			err := c.fetchAndSet(ctx, key, dest, ttl, fetch)
			c.releaseLock(ctx, key, token)
			return err
		}

		// 未拿到锁：等待持锁者写缓存后重读
		for i := 0; i < lockRetry; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(lockWaitTime):
			}
			found, err := c.Get(ctx, key, dest)
			if err != nil {
				return err
			}
			if found {
				return nil
			}
		}
		// 等待超时仍无缓存：本请求直接 fetch，避免雪崩下所有请求无限等待
		return c.fetchAndSet(ctx, key, dest, ttl, fetch)
	}
}

// acquireLock 获取 key 对应的防击穿锁，返回锁 token。
func (c *RedisCache) acquireLock(ctx context.Context, key string) (string, bool, error) {
	token := randomToken()
	ok, err := c.client.SetNX(ctx, c.lockKey(key), token, lockTTL).Result()
	if err != nil {
		return "", false, err
	}
	return token, ok, nil
}

// releaseLock 释放锁（仅当 token 匹配，防止误删他人锁）。
func (c *RedisCache) releaseLock(ctx context.Context, key, token string) {
	_ = releaseLockScript.Run(ctx, c.client, []string{c.lockKey(key)}, token).Err()
}

func (c *RedisCache) lockKey(key string) string {
	return c.prefix + ":" + lockPrefix + key
}

// fetchAndSet 执行 fetch 并写缓存。
func (c *RedisCache) fetchAndSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error {
	data, err := fetch()
	if err != nil {
		return err
	}
	if err := c.Set(ctx, key, data, ttl); err != nil {
		return err
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

// randomToken 生成锁 token
func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}
