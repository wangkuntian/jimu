//go:build jimu_noredis

package assembly

// Redis 在未编译 Redis 的构建里恒为 nil。
func (c *Context) Redis() any { return nil }

// Lock 在未编译 Redis 的构建里恒为 nil。
func (c *Context) Lock() any { return nil }
