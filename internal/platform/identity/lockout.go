package identity

import (
	"context"
	"fmt"
	"time"
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

// LoginFailureTracker 登录失败追踪与账号锁定。
// 存储由 FailureStore 抽象：Redis 支持跨实例共享锁定状态，内存实现只在单实例内有效。
type LoginFailureTracker struct {
	store FailureStore
	cfg   LockoutConfig
}

// NewLoginFailureTracker 创建登录失败追踪器
func NewLoginFailureTracker(store FailureStore, cfg LockoutConfig) *LoginFailureTracker {
	return &LoginFailureTracker{store: store, cfg: cfg}
}

// CheckLocked 检查账号是否被锁定
func (t *LoginFailureTracker) CheckLocked(ctx context.Context, username string) (bool, time.Duration, error) {
	if t.cfg.MaxFailures <= 0 {
		return false, 0, nil
	}
	key := lockKey(username)
	ttl, err := t.store.TTL(ctx, key)
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
	count, err := t.store.Incr(ctx, key, t.cfg.WindowSize, t.cfg.MaxFailures, t.cfg.LockoutDuration)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// Reset 清除失败计数（登录成功时调用）
func (t *LoginFailureTracker) Reset(ctx context.Context, username string) error {
	if t.cfg.MaxFailures <= 0 {
		return nil
	}
	return t.store.Del(ctx, lockKey(username))
}

func lockKey(username string) string {
	return fmt.Sprintf("jimu:auth:lockout:%s", username)
}
