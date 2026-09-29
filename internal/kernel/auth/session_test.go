package auth

import (
	"context"
	"testing"
	"time"
)

func TestSessionRotateRevokeAndRevokeAll(t *testing.T) {
	client := newFakeRedis()
	store := newRedisSessionStore(client)
	ctx := context.Background()

	if err := store.Create(ctx, 42, "session-1", "token-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.Rotate(ctx, 42, "session-1", "token-a", "token-b", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.Rotate(ctx, 42, "session-1", "token-a", "token-c", time.Minute); err != ErrTokenReuse {
		t.Fatalf("rotate reuse err = %v, want %v", err, ErrTokenReuse)
	}
	if err := store.Revoke(ctx, 42, "session-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Rotate(ctx, 42, "session-1", "token-b", "token-d", time.Minute); err != ErrSessionNotFound {
		t.Fatalf("rotate after revoke err = %v, want %v", err, ErrSessionNotFound)
	}

	if err := store.Create(ctx, 42, "session-2", "token-x", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, 7, "session-9", "token-y", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeAll(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if err := store.Rotate(ctx, 42, "session-2", "token-x", "token-z", time.Minute); err != ErrSessionNotFound {
		t.Fatalf("user revoke-all err = %v, want %v", err, ErrSessionNotFound)
	}
	if err := store.Rotate(ctx, 7, "session-9", "token-y", "token-z", time.Minute); err != nil {
		t.Fatalf("other user session broken: %v", err)
	}
}
