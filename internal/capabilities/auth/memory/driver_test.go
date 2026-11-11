package authmemory

import (
	"context"
	"testing"
	"time"

	"jimu/internal/capabilities/auth/authstore"
	"jimu/internal/platform/identity"

	"github.com/stretchr/testify/require"
)

func TestMemoryStoresSupportSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	stores, err := NewFromOptions(authstore.Options{ResetCodeTTL: time.Minute})
	require.NoError(t, err)

	require.NoError(t, stores.Sessions.Create(ctx, 42, "session-1", "token-1", time.Minute))
	require.ErrorIs(t, stores.Sessions.Rotate(ctx, 42, "session-1", "wrong-token", "token-2", time.Minute), identity.ErrTokenReuse)
	require.NoError(t, stores.Sessions.Rotate(ctx, 42, "session-1", "token-1", "token-2", time.Minute))
	require.NoError(t, stores.Sessions.Revoke(ctx, 42, "session-1"))
	require.ErrorIs(t, stores.Sessions.Rotate(ctx, 42, "session-1", "token-2", "token-3", time.Minute), identity.ErrSessionNotFound)
}

func TestMemoryStoresSupportOneTimeResetCode(t *testing.T) {
	ctx := context.Background()
	stores, err := NewFromOptions(authstore.Options{ResetCodeTTL: time.Minute})
	require.NoError(t, err)
	require.Equal(t, time.Minute, stores.ResetCodes.TTL())

	require.NoError(t, stores.ResetCodes.Set(ctx, "alice-hash", "123456"))
	code, err := stores.ResetCodes.GetAndDelete(ctx, "alice-hash")
	require.NoError(t, err)
	require.Equal(t, "123456", code)

	code, err = stores.ResetCodes.GetAndDelete(ctx, "alice-hash")
	require.NoError(t, err)
	require.Empty(t, code)
}
