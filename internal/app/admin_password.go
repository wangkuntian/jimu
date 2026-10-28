package app

import (
	"strings"

	"jimu/internal/config"
)

// AdminPassword returns the one-time structural seed password.
// File input is preferred so Compose can mount it without exposing the value in argv.
func AdminPassword() string {
	return strings.TrimSpace(config.GetEnvOrFile("ADMIN_PASSWORD_FILE", "ADMIN_PASSWORD"))
}
