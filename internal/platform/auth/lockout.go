package auth

import (
	"context"
	"fmt"
	"time"

	redistore "jimu/internal/platform/redis"
)

// LockoutConfig 账号锁定配置
type LockoutConfig struct {
	MaxFailures     int           // 连续失败次数阈值，0 表示不锁定
	LockoutDuration time.Duration // 锁定时长
	WindowSize      time.Duration // 计数窗口
}

// DefaultLockoutConfig 返回默认锁定配置
func DefaultLockoutConfig() LockoutConfig {
	return LockoutConfig{
		MaxFailures:     5,
		LockoutDuration: 15 * time.Minute,
		WindowSize:      15 * time.Minute,
	}
}

// LoginFailureTracker 基于 Redis 的登录失败追踪与账号锁定
type LoginFailureTracker struct {
	client redistore.Client
	cfg    LockoutConfig
}

// NewLoginFailureTracker 创建登录失败追踪器
func NewLoginFailureTracker(client redistore.Client, cfg LockoutConfig) *LoginFailureTracker {
	return &LoginFailureTracker{client: client, cfg: cfg}
}

// CheckLocked 检查账号是否被锁定
func (t *LoginFailureTracker) CheckLocked(ctx context.Context, username string) (bool, time.Duration, error) {
	if t.cfg.MaxFailures <= 0 {
		return false, 0, nil
	}
	key := lockKey(username)
	ttl, err := t.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if ttl <= 0 {
		return false, 0, nil
	}
	// 键存在且未过期 = 已锁定
	return true, ttl, nil
}

// RecordFailure 记录一次登录失败，返回当前连续失败次数
func (t *LoginFailureTracker) RecordFailure(ctx context.Context, username string) (int, error) {
	if t.cfg.MaxFailures <= 0 {
		return 0, nil
	}
	key := lockKey(username)
	count, err := t.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		_ = t.client.Expire(ctx, key, t.cfg.WindowSize)
	}
	// 达到阈值，延长锁定时间为 LockoutDuration
	if int(count) >= t.cfg.MaxFailures {
		if err := t.client.Expire(ctx, key, t.cfg.LockoutDuration).Err(); err != nil {
			return int(count), err
		}
	}
	return int(count), nil
}

// Reset 清除失败计数（登录成功时调用）
func (t *LoginFailureTracker) Reset(ctx context.Context, username string) error {
	if t.cfg.MaxFailures <= 0 {
		return nil
	}
	return t.client.Del(ctx, lockKey(username)).Err()
}

func lockKey(username string) string {
	return fmt.Sprintf("jimu:auth:lockout:%s", username)
}
