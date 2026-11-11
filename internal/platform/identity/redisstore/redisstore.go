// Package redisstore 提供 auth 存储抽象的 Redis 实现。
//
// 单独成包是为了让 platform/identity 主包不 import go-redis：无 Redis 的项目
// 只编内存实现，Redis 实现按需引入。
package redisstore

import (
	"context"
	"time"

	"jimu/internal/platform/identity"
	redistore "jimu/internal/platform/redisclient"
)

// RateCounter 基于 Redis 的限流计数器（跨实例精确）。
type RateCounter struct{ client redistore.Client }

// NewRateCounter 返回 Redis 实现的限流计数器。
func NewRateCounter(client redistore.Client) identity.RateCounter {
	return &RateCounter{client: client}
}

// Increment 实现 identity.RateCounter。
func (r *RateCounter) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	current, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if current == 1 {
		if err := r.client.PExpire(ctx, key, window).Err(); err != nil {
			return current, err
		}
	}
	return current, nil
}

// FailureStore 基于 Redis 的登录失败计数（跨实例共享锁定状态）。
type FailureStore struct{ client redistore.Client }

// NewFailureStore 返回 Redis 实现的失败计数存储。
func NewFailureStore(client redistore.Client) identity.FailureStore {
	return &FailureStore{client: client}
}

// Incr 实现 identity.FailureStore。
func (f *FailureStore) Incr(ctx context.Context, key string, window time.Duration, threshold int, lockoutTTL time.Duration) (int64, error) {
	count, err := f.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		if err := f.client.Expire(ctx, key, window).Err(); err != nil {
			return count, err
		}
	}
	if threshold > 0 && int(count) >= threshold {
		if err := f.client.Expire(ctx, key, lockoutTTL).Err(); err != nil {
			return count, err
		}
	}
	return count, nil
}

// Get 实现 identity.FailureStore。
func (f *FailureStore) Get(ctx context.Context, key string) (int64, error) {
	value, err := f.client.Get(ctx, key).Int64()
	if err != nil {
		return 0, nil
	}
	return value, nil
}

// TTL 实现 identity.FailureStore。
func (f *FailureStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return f.client.TTL(ctx, key).Result()
}

// Del 实现 identity.FailureStore。
func (f *FailureStore) Del(ctx context.Context, key string) error {
	return f.client.Del(ctx, key).Err()
}

// NewSessionStore 返回 Redis 实现的会话存储（跨实例共享会话与强制下线）。
func NewSessionStore(client redistore.Client) identity.SessionStore {
	return NewRedisSessionStore(client)
}
