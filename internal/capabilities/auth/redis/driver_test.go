package authredis

import (
	"testing"

	"jimu/internal/capabilities/auth/authstore"

	"github.com/stretchr/testify/require"
)

func TestNewFromOptionsRequiresRedisClient(t *testing.T) {
	_, err := NewFromOptions(authstore.Options{})
	require.ErrorContains(t, err, "redis client is not available")
}
