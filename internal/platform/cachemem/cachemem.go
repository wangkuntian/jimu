// Package cachemem 提供进程内的 contract.Cache 实现。
//
// 它属于平台层而不是某个能力：需要「缓存不可用时降级到本地」的能力都可以直接用它，
// 不必依赖 cache 能力（能力之间禁止互相 import）。单实例语义，多实例部署不共享。
package cachemem

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"jimu/internal/contract"
)

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

// Get 实现 contract.Cache。
func (c *Cache) Get(_ context.Context, key string, dest any) (bool, error) {
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

// Set 实现 contract.Cache；ttl <= 0 表示不过期。
func (c *Cache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
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

// Delete 实现 contract.Cache。
func (c *Cache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, c.key(key))
	c.mu.Unlock()
	return nil
}

// DeletePattern 删除匹配前缀的 key（只处理结尾的 `*`，与既有调用方式一致）。
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

// Exists 判断 key 存在且未过期。
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

// GetOrSet 未命中时回源并写回。
func (c *Cache) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fetch func() (any, error)) error {
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

// Close 无资源需要释放。
func (c *Cache) Close() error { return nil }

var _ contract.Cache = (*Cache)(nil)
