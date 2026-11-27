//go:build !jimu_lean

package assembly

import (
	"jimu/internal/platform/httpclient"
	"jimu/internal/platform/reporter"
	"jimu/internal/platform/scheduler"
)

// HTTPClient 统一出站 HTTP client。
func (c *Context) HTTPClient() *httpclient.Client {
	client, _ := c.container.HTTPClient.(*httpclient.Client)
	return client
}

// Scheduler 定时任务调度器。
func (c *Context) Scheduler() *scheduler.CronScheduler {
	sched, _ := c.container.Scheduler.(*scheduler.CronScheduler)
	return sched
}

// Reporter 错误上报器。
func (c *Context) Reporter() reporter.Reporter {
	rep, _ := c.container.Reporter.(reporter.Reporter)
	return rep
}
