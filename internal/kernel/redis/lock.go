package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// Lock 基于 Redis 的分布式锁
type Lock struct {
	client Client
	prefix string
}

// NewLock 创建分布式锁实例
func NewLock(client Client, prefix string) *Lock {
	if prefix == "" {
		prefix = "lock"
	}
	return &Lock{
		client: client,
		prefix: prefix,
	}
}

// AcquireResult 锁获取结果
type AcquireResult struct {
	Key     string // 锁的 Redis key
	Token   string // 锁的唯一标识（用于释放）
	Expires time.Time
}

// Acquire 获取锁
// key: 锁名称
// ttl: 锁有效期（必须设置，防止死锁）
// retry: 重试次数，0 表示不重试
// retryInterval: 重试间隔
func (l *Lock) Acquire(ctx context.Context, key string, ttl time.Duration, retry int, retryInterval time.Duration) (*AcquireResult, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("lock TTL must be positive")
	}

	redisKey := l.prefix + ":" + key
	token := generateToken()

	for attempt := 0; attempt <= retry; attempt++ {
		ok, err := l.client.SetNX(ctx, redisKey, token, ttl).Result()
		if err != nil {
			return nil, fmt.Errorf("redis setnx: %w", err)
		}
		if ok {
			return &AcquireResult{
				Key:     redisKey,
				Token:   token,
				Expires: time.Now().Add(ttl),
			}, nil
		}

		if attempt < retry && retryInterval > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryInterval):
				// continue to next attempt
			}
		}
	}

	return nil, fmt.Errorf("failed to acquire lock %q after %d attempts", key, retry+1)
}

// Release 释放锁（仅当 token 匹配时才释放，防止误释放他人的锁）
func (l *Lock) Release(ctx context.Context, result *AcquireResult) error {
	if result == nil {
		return nil
	}

	// 使用 Lua 脚本保证原子性：只有 token 匹配时才删除
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	_, err := l.client.Eval(ctx, script, []string{result.Key}, result.Token).Result()
	if err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

// Extend 延长锁有效期（仅当 token 匹配时）
func (l *Lock) Extend(ctx context.Context, result *AcquireResult, ttl time.Duration) error {
	if result == nil {
		return nil
	}

	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	else
		return 0
	end
	`

	ok, err := l.client.Eval(ctx, script, []string{result.Key}, result.Token, ttl.Milliseconds()).Bool()
	if err != nil {
		return fmt.Errorf("extend lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("lock %q expired or token mismatch", result.Key)
	}
	return nil
}

// TryAcquire 尝试获取锁（不重试）
func (l *Lock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (*AcquireResult, error) {
	return l.Acquire(ctx, key, ttl, 0, 0)
}

// WithLock 在锁保护下执行函数
func (l *Lock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	result, err := l.TryAcquire(ctx, key, ttl)
	if err != nil {
		return err
	}
	defer func() { _ = l.Release(ctx, result) }()
	return fn()
}

// generateToken 生成唯一 token
func generateToken() string {
	b := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, b)
	return hex.EncodeToString(b)
}
