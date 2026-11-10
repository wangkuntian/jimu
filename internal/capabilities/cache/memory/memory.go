// Package memory 是 cache 能力的进程内驱动。
//
// 单实例语义：每个进程各自持有一份缓存，多实例部署时实例间不共享。
// 适合 0 依赖项目和测试；需要跨实例一致性的场景应选 redis 驱动。
package memory

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"jimu/internal/capabilities/cache"
)

func init() { cache.Register("memory", newFromOptions) }

func newFromOptions(opts cache.Options) (cache.Cache, error) {
	return New(opts.Prefix), nil
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// Cache 是进程内缓存实现。
type Cache struct {
	mu     sync.RWMutex
	items  map[string]entry
	prefix string
}

// New 创建进程内缓存；prefix 为空时不加前缀。
func New(prefix string) *Cache {
	return &Cache{items: map[string]entry{}, prefix: prefix}
}

func (c *Cache) key(k string) string {
	if c.prefix == "" {
		return k
	}
	return c.prefix + ":" + k
}

func (c *Cache) Get(_ context.Context, key string, dest interface{}) (bool, error) {
	c.mu.RLock()
	item, ok := c.items[c.key(key)]
	c.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, c.key(key))
		c.mu.Unlock()
		return false, nil
	}
	if err := json.Unmarshal(item.value, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[c.key(key)] = entry{value: data, expiresAt: expires}
	c.mu.Unlock()
	return nil
}

func (c *Cache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, c.key(key))
	c.mu.Unlock()
	return nil
}

// DeletePattern 支持 Redis 风格的 `*` 通配（只处理结尾的 `*`，与既有调用方式一致）。
func (c *Cache) DeletePattern(_ context.Context, pattern string) error {
	full := c.key(pattern)
	prefix, isGlob := strings.CutSuffix(full, "*")
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if isGlob && strings.HasPrefix(k, prefix) || !isGlob && k == full {
			delete(c.items, k)
		}
	}
	return nil
}

func (c *Cache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, ok := c.items[c.key(key)]
	c.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error {
	if ok, err := c.Get(ctx, key, dest); err != nil {
		return err
	} else if ok {
		return nil
	}
	value, err := fetch()
	if err != nil {
		return err
	}
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *Cache) Close() error { return nil }

var _ cache.Cache = (*Cache)(nil)
