//go:build !jimu_noredis

package assembly

import (
	redistore "jimu/internal/platform/redisclient"
)

// Redis 缓存客户端。
func (c *Context) Redis() redistore.Client { return c.container.RedisClient() }

// Lock 分布式锁（多实例协调）。
func (c *Context) Lock() *redistore.Lock { return c.container.RedisLock() }
