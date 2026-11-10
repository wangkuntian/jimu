// Package lock 定义进程间互斥锁的最小接口。
//
// 单独成包是为了让 scheduler 不 import kernel/redis：只要 scheduler 直接引用
// redis.Lock，0 能力项目也会把整个 go-redis 编进二进制。
package lock

import (
	"context"
	"time"
)

// Result 是一次成功加锁的凭据；释放时必须原样传回。
type Result struct {
	Key     string
	Token   string
	Expires time.Time
}

// Locker 是分布式锁的最小面：只覆盖调度器需要的「尝试加锁 / 释放」。
// 单实例部署传 nil 即可（不加锁）。
type Locker interface {
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (*Result, error)
	Release(ctx context.Context, result *Result) error
}
