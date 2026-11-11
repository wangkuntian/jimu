package authmemory

import (
	"context"
	"sync"
	"time"
)

type resetEntry struct {
	code    string
	expires time.Time
}

// ResetStore 是密码重置验证码的进程内一次性存储。
type ResetStore struct {
	mu      sync.Mutex
	entries map[string]resetEntry
	ttl     time.Duration
}

// NewResetStore 创建进程内重置码存储。
func NewResetStore(ttl time.Duration) *ResetStore {
	return &ResetStore{entries: map[string]resetEntry{}, ttl: ttl}
}

// Set 存储验证码。
func (s *ResetStore) Set(_ context.Context, emailHash, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[emailHash] = resetEntry{code: code, expires: time.Now().Add(s.ttl)}
	return nil
}

// GetAndDelete 原子读取并删除验证码。
func (s *ResetStore) GetAndDelete(_ context.Context, emailHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[emailHash]
	delete(s.entries, emailHash)
	if !ok || time.Now().After(entry.expires) {
		return "", nil
	}
	return entry.code, nil
}

// TTL 返回验证码有效期。
func (s *ResetStore) TTL() time.Duration { return s.ttl }
