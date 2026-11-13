package tui

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAvailableRequiresBothTTYsAndHonorsDisable(t *testing.T) {
	t.Setenv(noTUIEnv, "")
	require.False(t, Available(bytes.NewBuffer(nil), &bytes.Buffer{}))
	require.True(t, available(true, true, false))
	require.False(t, available(true, false, false))
	require.False(t, available(true, true, true))

	t.Setenv(noTUIEnv, "1")
	require.False(t, available(true, true, false))
}
