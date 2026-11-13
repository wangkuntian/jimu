//go:build jimu_lean

package assembly

import (
	"jimu/internal/contract"
)

// HTTPClient 在 lean 构建里没有出站客户端；未选中任何消费者时不会被调用。
func (c *Context) HTTPClient() contract.HTTPClient { return c.container.HTTPClient }

// Scheduler 在 lean 构建里没有调度器。
func (c *Context) Scheduler() contract.Scheduler { return c.container.Scheduler }

// Reporter 在 lean 构建里没有错误上报器。
func (c *Context) Reporter() contract.Reporter { return c.container.Reporter }
