package auth

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrSessionNotFound 会话不存在（已登出或过期）。
	ErrSessionNotFound = errors.New("session not found")
	// ErrTokenReuse 检测到 refresh token 复用（可能是凭证泄露）。
	ErrTokenReuse = errors.New("refresh token reuse detected")
)

// SessionStore 是会话状态的存储抽象。
// Redis 实现支持跨实例共享与强制下线；内存实现只在单实例内有效。
type SessionStore interface {
	Create(ctx context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error
	Rotate(ctx context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, ttl time.Duration) error
	Revoke(ctx context.Context, userID uint64, sessionID string) error
	RevokeAll(ctx context.Context, userID uint64) error
}

// RateCounter 是限流计数器的存储抽象。
// Redis 实现提供跨实例的精确计数；内存实现只在单实例内准确。
type RateCounter interface {
	// Increment 在窗口内自增一次计数并返回当前值。
	// window 只在首次自增（返回值 1）时生效。
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
}

// FailureStore 是登录失败计数的存储抽象。
type FailureStore interface {
	// Incr 自增失败计数并返回当前值；首次创建时用 window 设定过期时间，
	// 计数达到 threshold 后改用 lockoutTTL（在剩余时间小于它时延长）。
	Incr(ctx context.Context, key string, window time.Duration, threshold int, lockoutTTL time.Duration) (int64, error)
	// Get 读取当前失败计数；不存在时返回 0。
	Get(ctx context.Context, key string) (int64, error)
	// TTL 返回剩余锁定时长；不存在或已过期返回 0。
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Del 清除失败计数（登录成功或管理员解锁时调用）。
	Del(ctx context.Context, key string) error
}

// memoryCounter 是 RateCounter 的进程内实现：单实例准确，多实例各自计数。
type memoryCounter struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
}

type memoryEntry struct {
	count   int64
	expires time.Time
}

// NewMemoryRateCounter 返回进程内限流计数器。
// 多实例部署时每个实例独立计数，实际限流阈值约为配置值 × 实例数。
func NewMemoryRateCounter() RateCounter {
	return &memoryCounter{entries: map[string]memoryEntry{}}
}

func (m *memoryCounter) Increment(_ context.Context, key string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	entry, ok := m.entries[key]
	if !ok || now.After(entry.expires) {
		entry = memoryEntry{expires: now.Add(window)}
	}
	entry.count++
	m.entries[key] = entry
	return entry.count, nil
}

// memoryFailures 是 FailureStore 的进程内实现：重启即清空，适合单实例或无 Redis 场景。
type memoryFailures struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
}

// NewMemoryFailureStore 返回进程内失败计数存储。
// 进程重启会清空锁定状态；多实例部署时各实例独立计数。
func NewMemoryFailureStore() FailureStore {
	return &memoryFailures{entries: map[string]memoryEntry{}}
}

func (m *memoryFailures) Incr(_ context.Context, key string, window time.Duration, threshold int, lockoutTTL time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	entry, ok := m.entries[key]
	if !ok || now.After(entry.expires) {
		entry = memoryEntry{expires: now.Add(window)}
	}
	entry.count++
	if threshold > 0 && entry.count >= int64(threshold) {
		if locked := now.Add(lockoutTTL); locked.After(entry.expires) {
			entry.expires = locked
		}
	}
	m.entries[key] = entry
	return entry.count, nil
}

func (m *memoryFailures) Get(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[key]
	if !ok || time.Now().After(entry.expires) {
		return 0, nil
	}
	return entry.count, nil
}

func (m *memoryFailures) TTL(_ context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[key]
	if !ok {
		return 0, nil
	}
	remaining := time.Until(entry.expires)
	if remaining <= 0 {
		return 0, nil
	}
	return remaining, nil
}

func (m *memoryFailures) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}
