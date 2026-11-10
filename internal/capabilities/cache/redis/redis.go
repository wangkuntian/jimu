// Package redis 是 cache 能力的 Redis 驱动。
package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"jimu/internal/capabilities/cache"
	redistore "jimu/internal/kernel/redis"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

func init() { cache.Register("redis", newFromOptions) }

// newFromOptions 把装配期注入的 Redis 客户端包装成 cache.Cache。
func newFromOptions(opts cache.Options) (cache.Cache, error) {
	client, ok := opts.Client.(redistore.Client)
	if !ok || client == nil {
		return nil, fmt.Errorf("cache driver \"redis\" requires a redis client (got %T)", opts.Client)
	}
	return newCache(client, opts.Prefix), nil
}

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

// Cache 基于 Redis 的缓存实现
type Cache struct {
	client redistore.Client
	prefix string
	sf     singleflight.Group // 进程内合并同 key 并发回源
}

// NewCache 创建 Redis 缓存
func newCache(client redistore.Client, prefix string) *Cache {
	return &Cache{
		client: client,
		prefix: prefix,
	}
}

// key 生成带前缀的 key
func (c *Cache) key(k string) string {
	return fmt.Sprintf("%s:%s", c.prefix, k)
}

// Get 获取缓存
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
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
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete 删除缓存
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

// DeletePattern 按模式删除缓存
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
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
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

// GetOrSet 缓存不存在时自动获取并设置（Cache-Aside 模式）。
// 防击穿分两级：
//  1. 进程内 singleflight：同一实例内同 key 并发未命中只回源一次；
//  2. Redis 分布式锁：跨实例并发未命中只有一个回源，其余等待重读（锁带 TTL 防死锁，
//     等待超时则直接回源兜底，不阻塞调用方）。
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error {
	found, err := c.Get(ctx, key, dest)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// singleflight 结果可能是原始 JSON（命中缓存）或回源数据
	v, err, _ := c.sf.Do(c.key(key), func() (interface{}, error) {
		// 进程内二次确认：leader 可能已回填缓存
		var raw json.RawMessage
		if ok, err := c.Get(ctx, key, &raw); err == nil && ok {
			return raw, nil
		}
		return c.fetchWithLock(ctx, key, ttl, fetch)
	})
	if err != nil {
		return err
	}
	return assign(v, dest)
}

// fetchWithLock 跨实例防击穿：抢到锁者回源并写缓存，其余等待重读，超时兜底回源。
func (c *Cache) fetchWithLock(ctx context.Context, key string, ttl time.Duration, fetch func() (interface{}, error)) (interface{}, error) {
	token, ok, err := c.acquireLock(ctx, key)
	if err != nil {
		// 锁服务异常时直接回源，保证可用性
		return c.fetchAndSet(ctx, key, ttl, fetch)
	}
	if ok {
		data, err := c.fetchAndSet(ctx, key, ttl, fetch)
		c.releaseLock(ctx, key, token)
		return data, err
	}

	// 未拿到锁：等待持锁者写缓存后重读
	for i := 0; i < lockRetry; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockWaitTime):
		}
		var raw json.RawMessage
		ok, err := c.Get(ctx, key, &raw)
		if err != nil {
			return nil, err
		}
		if ok {
			return raw, nil
		}
	}
	// 等待超时仍无缓存：本请求直接回源，避免雪崩下所有请求无限等待
	return c.fetchAndSet(ctx, key, ttl, fetch)
}

// acquireLock 获取 key 对应的防击穿锁，返回锁 token。
func (c *Cache) acquireLock(ctx context.Context, key string) (string, bool, error) {
	token := randomToken()
	ok, err := c.client.SetNX(ctx, c.lockKey(key), token, lockTTL).Result()
	if err != nil {
		return "", false, err
	}
	return token, ok, nil
}

// releaseLock 释放锁（仅当 token 匹配，防止误删他人锁）。
func (c *Cache) releaseLock(ctx context.Context, key, token string) {
	_ = releaseLockScript.Run(ctx, c.client, []string{c.lockKey(key)}, token).Err()
}

func (c *Cache) lockKey(key string) string {
	return c.prefix + ":" + lockPrefix + key
}

// fetchAndSet 执行 fetch 并写缓存，返回待反序列化的数据。
func (c *Cache) fetchAndSet(ctx context.Context, key string, ttl time.Duration, fetch func() (interface{}, error)) (interface{}, error) {
	data, err := fetch()
	if err != nil {
		return nil, err
	}
	if err := c.Set(ctx, key, data, ttl); err != nil {
		return nil, err
	}
	return data, nil
}

// assign 把回源结果（业务值或原始 JSON）写入调用方的 dest。
func assign(v interface{}, dest interface{}) error {
	bytes, err := json.Marshal(v)
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
func (c *Cache) Close() error {
	return c.client.Close()
}
