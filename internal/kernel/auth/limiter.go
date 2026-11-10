package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Limiter struct {
	counter    RateCounter
	failClosed bool
}

// NewLimiter 用任意 RateCounter 构造限流器：Redis 提供跨实例精确计数，
// 内存实现只在单实例内准确。
func NewLimiter(counter RateCounter, failClosed bool) *Limiter {
	return &Limiter{counter: counter, failClosed: failClosed}
}

func (l *Limiter) Allow(ctx context.Context, scope, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	current, err := l.counter.Increment(ctx, LimitKey(scope, key), window)
	if err != nil {
		return !l.failClosed, err
	}
	return current <= int64(limit), nil
}

// LimitKey 返回限流计数器在 Redis 中的键名（sha256(key) 避免明文泄露）。
// 导出供运维 peek 端点复用，保证查询与限流器写入同一 key。
func LimitKey(scope, key string) string {
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("jimu:auth:limit:%s:%s", scope, hex.EncodeToString(sum[:]))
}
