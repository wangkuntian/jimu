package auth

import (
	"context"
	"testing"
	"time"
)

// TestMemoryRateCounterCountsWithinWindow 钉住窗口内计数与窗口外重置。
func TestMemoryRateCounterCountsWithinWindow(t *testing.T) {
	counter := NewMemoryRateCounter()
	ctx := context.Background()

	for want := int64(1); want <= 3; want++ {
		got, err := counter.Increment(ctx, "k", time.Minute)
		if err != nil {
			t.Fatalf("Increment: %v", err)
		}
		if got != want {
			t.Fatalf("Increment = %d, want %d", got, want)
		}
	}

	// 窗口过期后重新从 1 开始计数。
	fresh := NewMemoryRateCounter()
	if got, _ := fresh.Increment(ctx, "k", time.Nanosecond); got != 1 {
		t.Fatalf("first Increment = %d, want 1", got)
	}
	time.Sleep(2 * time.Millisecond)
	if got, _ := fresh.Increment(ctx, "k", time.Nanosecond); got != 1 {
		t.Fatalf("Increment after window expiry = %d, want 1 (new window)", got)
	}
}

// TestMemoryFailureStoreLocksAtThreshold 钉住「达到阈值后延长锁定」的语义。
func TestMemoryFailureStoreLocksAtThreshold(t *testing.T) {
	store := NewMemoryFailureStore()
	ctx := context.Background()

	window := time.Minute
	lockout := time.Hour
	for i := 1; i <= 3; i++ {
		count, err := store.Incr(ctx, "u", window, 3, lockout)
		if err != nil {
			t.Fatalf("Incr: %v", err)
		}
		if count != int64(i) {
			t.Fatalf("count = %d, want %d", count, i)
		}
	}

	ttl, err := store.TTL(ctx, "u")
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl < 59*time.Minute {
		t.Fatalf("TTL = %v, want close to %v (lockout must extend past window)", ttl, lockout)
	}

	if err := store.Del(ctx, "u"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	count, err := store.Get(ctx, "u")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if count != 0 {
		t.Fatalf("Get after Del = %d, want 0", count)
	}
}
