package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/app"

	"github.com/stretchr/testify/require"
)

func TestAdminPasswordPrefersFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin_password")
	require.NoError(t, os.WriteFile(path, []byte("from-file\n"), 0o600))
	t.Setenv("ADMIN_PASSWORD_FILE", path)
	t.Setenv("ADMIN_PASSWORD", "from-env")
	require.Equal(t, "from-file", app.AdminPassword())
}

func TestAdminPasswordFallsBackToEnv(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "from-env")
	require.Equal(t, "from-env", app.AdminPassword())
}

func TestAdminPasswordMissing(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "")
	require.Empty(t, app.AdminPassword())
}

func TestAdminPasswordTrimsDirectEnv(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "  from-env\n")
	require.Equal(t, "from-env", app.AdminPassword())
}
