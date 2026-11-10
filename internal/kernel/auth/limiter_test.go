package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// recordingCounter 记录被自增的 key，用于断言 key 不含明文身份信息。
type recordingCounter struct {
	RateCounter
	keys []string
}

func (r *recordingCounter) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	r.keys = append(r.keys, key)
	return r.RateCounter.Increment(ctx, key, window)
}

// failingCounter 模拟存储不可用。
type failingCounter struct{}

func (failingCounter) Increment(context.Context, string, time.Duration) (int64, error) {
	return 0, errors.New("redis down")
}

func TestLimiterAllowsOnlyConfiguredWindow(t *testing.T) {
	limiter, closeServer := newTestLimiter(t, true)
	defer closeServer()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 5, time.Minute)
		if err != nil || !ok {
			t.Fatalf("attempt %d: ok=%v err=%v", i, ok, err)
		}
	}
	ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 5, time.Minute)
	if err != nil || ok {
		t.Fatalf("sixth attempt: ok=%v err=%v", ok, err)
	}
}

func TestLimiterScopesAreIsolated(t *testing.T) {
	limiter, closeServer := newTestLimiter(t, true)
	defer closeServer()
	ctx := context.Background()

	if ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 1, time.Minute); err != nil || !ok {
		t.Fatalf("login first: ok=%v err=%v", ok, err)
	}
	if ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 1, time.Minute); err != nil || ok {
		t.Fatalf("login second: ok=%v err=%v", ok, err)
	}
	if ok, err := limiter.Allow(ctx, "register", "ip:127.0.0.1", 1, time.Minute); err != nil || !ok {
		t.Fatalf("register isolated: ok=%v err=%v", ok, err)
	}
}

func TestLimiterFailsClosedInProductionMode(t *testing.T) {
	limiter := NewLimiter(failingCounter{}, true)

	ok, err := limiter.Allow(context.Background(), "login", "ip:127.0.0.1", 5, time.Minute)
	if err == nil || ok {
		t.Fatalf("closed redis: ok=%v err=%v", ok, err)
	}
}

func TestLimiterDoesNotExposeUsernameInRedisKeys(t *testing.T) {
	counter := &recordingCounter{RateCounter: NewMemoryRateCounter()}
	limiter := NewLimiter(counter, true)

	if ok, err := limiter.Allow(context.Background(), "login", "username:alice@example.com", 5, time.Minute); err != nil || !ok {
		t.Fatalf("allow: ok=%v err=%v", ok, err)
	}
	for _, key := range counter.keys {
		if strings.Contains(key, "alice") || strings.Contains(key, "example.com") {
			t.Fatalf("redis key leaked username: %s", key)
		}
	}
}

func newTestLimiter(t *testing.T, failClosed bool) (*Limiter, func()) {
	t.Helper()
	return NewLimiter(NewMemoryRateCounter(), failClosed), func() {}
}
