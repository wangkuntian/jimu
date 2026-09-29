package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

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
	client := newFakeRedis()
	client.err = errors.New("redis down")
	limiter := NewLimiter(client, true)

	ok, err := limiter.Allow(context.Background(), "login", "ip:127.0.0.1", 5, time.Minute)
	if err == nil || ok {
		t.Fatalf("closed redis: ok=%v err=%v", ok, err)
	}
}

func TestLimiterDoesNotExposeUsernameInRedisKeys(t *testing.T) {
	client := newFakeRedis()
	limiter := NewLimiter(client, true)

	if ok, err := limiter.Allow(context.Background(), "login", "username:alice@example.com", 5, time.Minute); err != nil || !ok {
		t.Fatalf("allow: ok=%v err=%v", ok, err)
	}
	for _, key := range client.keys() {
		if strings.Contains(key, "alice") || strings.Contains(key, "example.com") {
			t.Fatalf("redis key leaked username: %s", key)
		}
	}
}

func newTestLimiter(t *testing.T, failClosed bool) (*Limiter, func()) {
	t.Helper()
	return NewLimiter(newFakeRedis(), failClosed), func() {}
}
